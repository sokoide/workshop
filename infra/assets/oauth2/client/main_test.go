package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestApp(tokenURL, apiURL string) *app {
	return &app{
		clientID:     "workshop-client",
		redirectURI:  "http://localhost:3000/callback",
		issuerURL:    "http://issuer.example.com",
		authURL:      "http://issuer.example.com/auth",
		tokenURL:     tokenURL,
		apiURL:       apiURL,
		scope:        "openid profile email",
		sessionStore: make(map[string]session),
	}
}

func TestPKCEChallenge(t *testing.T) {
	// RFC 7636 Appendix B.2 の参考ベクトル。
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if got := pkceChallenge(verifier); got != want {
		t.Errorf("pkceChallenge(%q) = %q, want %q", verifier, got, want)
	}
}

func TestRandomURLSafe(t *testing.T) {
	a, err := randomURLSafe(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := randomURLSafe(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RawURL エンコード: 32 バイト → 43 文字、パディングなし。
	if len(a) != base64.RawURLEncoding.EncodedLen(32) {
		t.Errorf("unexpected encoded length %d", len(a))
	}
	if a == b {
		t.Error("two random values should differ")
	}
}

func TestGetenv(t *testing.T) {
	t.Run("returns the environment value when set", func(t *testing.T) {
		t.Setenv("OAUTH_CLIENT_ID", "other-client")

		if got := getenv("OAUTH_CLIENT_ID", "fallback"); got != "other-client" {
			t.Errorf("expected the env value, got %q", got)
		}
	})

	t.Run("falls back when unset or blank", func(t *testing.T) {
		t.Setenv("OAUTH_CLIENT_ID", " ")

		if got := getenv("OAUTH_CLIENT_ID", "fallback"); got != "fallback" {
			t.Errorf("expected the fallback, got %q", got)
		}
	})
}

func TestHTMLEscape(t *testing.T) {
	got := htmlEscape(`<a href="x">&'`)
	if want := `&lt;a href=&quot;x&quot;&gt;&amp;&#39;`; got != want {
		t.Errorf("htmlEscape = %q, want %q", got, want)
	}
}

func TestSessionStore(t *testing.T) {
	a := newTestApp("http://token.example.com", "http://api.example.com")

	if _, ok := a.getSession("no-such"); ok {
		t.Error("expected a missing session")
	}

	sess := session{State: "st", CodeVerifier: "v", Expiry: time.Now().Add(time.Minute)}
	a.setSession("sid", sess)

	got, ok := a.getSession("sid")
	if !ok || got.State != "st" || got.CodeVerifier != "v" {
		t.Errorf("unexpected session: %+v (found=%t)", got, ok)
	}
}

func TestHandleLogin(t *testing.T) {
	a := newTestApp("http://token.example.com", "http://api.example.com")

	rec := httptest.NewRecorder()
	a.handleLogin(rec, httptest.NewRequest(http.MethodGet, "/login", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}

	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid Location header: %v", err)
	}
	q := location.Query()
	if q.Get("client_id") != "workshop-client" ||
		q.Get("response_type") != "code" ||
		q.Get("code_challenge_method") != "S256" ||
		q.Get("code_challenge") == "" || q.Get("state") == "" {
		t.Errorf("authorization request is missing required parameters: %s", location.RawQuery)
	}

	// state と PKCE verifier が cookie に紐づくセッションに保存されていること。
	cookie := parseSessionCookie(t, rec)
	sess, ok := a.getSession(cookie)
	if !ok {
		t.Fatal("login did not create a session")
	}
	if sess.State != q.Get("state") {
		t.Errorf("session state %q does not match redirect state %q", sess.State, q.Get("state"))
	}
	if sess.CodeVerifier == "" {
		t.Error("session is missing the PKCE code verifier")
	}
	if pkceChallenge(sess.CodeVerifier) != q.Get("code_challenge") {
		t.Error("code_challenge does not match the stored verifier")
	}
}

func TestHandleCallbackValidation(t *testing.T) {
	a := newTestApp("http://token.example.com", "http://api.example.com")

	tests := []struct {
		name    string
		prepare func() *http.Request
		want    string
	}{
		{
			name: "authorization error is surfaced",
			prepare: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/callback?error=access_denied", nil)
			},
			want: "Authorization failed: access_denied",
		},
		{
			name: "missing session",
			prepare: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/callback?code=c&state=s", nil)
			},
			want: "Session not found",
		},
		{
			name: "expired session",
			prepare: func() *http.Request {
				a.setSession("expired", session{State: "s", Expiry: time.Now().Add(-time.Minute)})
				req := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=s", nil)
				req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "expired"})
				return req
			},
			want: "Session expired",
		},
		{
			name: "state mismatch",
			prepare: func() *http.Request {
				a.setSession("hijack", session{State: "expected", Expiry: time.Now().Add(time.Minute)})
				req := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=evil", nil)
				req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "hijack"})
				return req
			},
			want: "Invalid state",
		},
		{
			name: "missing code",
			prepare: func() *http.Request {
				a.setSession("nocode", session{State: "s", Expiry: time.Now().Add(time.Minute)})
				req := httptest.NewRequest(http.MethodGet, "/callback?state=s", nil)
				req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "nocode"})
				return req
			},
			want: "Missing code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			a.handleCallback(rec, tt.prepare())

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
			if body := strings.TrimSpace(rec.Body.String()); body != tt.want {
				t.Errorf("expected %q, got %q", tt.want, body)
			}
		})
	}
}

func TestHandleCallbackExchangesCodeAndCallsAPI(t *testing.T) {
	// トークンエンドポイントとリソース API を httptest で立てる。
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("token request: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" ||
			r.Form.Get("code_verifier") != "stored-verifier" {
			t.Errorf("token request is missing PKCE parameters: %v", r.Form)
		}
		_ = json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "at-123",
			TokenType:   "Bearer",
			ExpiresIn:   300,
			IDToken:     "id-123",
			Scope:       "openid",
		})
	}))
	defer tokenSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer at-123" {
			t.Errorf("API call missing bearer token, got %q", got)
		}
		_, _ = w.Write([]byte(`{"message":"Hello, authenticated SWE!","sub":"user-1"}`))
	}))
	defer apiSrv.Close()

	a := newTestApp(tokenSrv.URL, apiSrv.URL)
	a.setSession("sid", session{
		State:        "known-state",
		CodeVerifier: "stored-verifier",
		Expiry:       time.Now().Add(time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, "/callback?code=abc&state=known-state", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid"})

	rec := httptest.NewRecorder()
	a.handleCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var payload struct {
		TokenReceived bool          `json:"token_received"`
		APIResult     apiCallResult `json:"api_result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid callback response: %v", err)
	}
	if !payload.TokenReceived {
		t.Error("callback should report token_received=true")
	}
	if payload.APIResult.StatusCode != http.StatusOK || !strings.Contains(payload.APIResult.Body, "authenticated") {
		t.Errorf("unexpected API result: %+v", payload.APIResult)
	}

	// トークンがセッションにキャッシュされ、/call-api で再利用されること。
	a.setSession("sid", mustGetSession(t, a, "sid"))
	callRec := httptest.NewRecorder()
	callReq := httptest.NewRequest(http.MethodGet, "/call-api", nil)
	callReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid"})
	a.handleCallAPI(callRec, callReq)

	if callRec.Code != http.StatusOK {
		t.Errorf("expected 200 on /call-api, got %d", callRec.Code)
	}
}

func TestHandleCallAPIRequiresToken(t *testing.T) {
	a := newTestApp("http://token.example.com", "http://api.example.com")
	a.setSession("sid", session{Expiry: time.Now().Add(time.Minute)})

	req := httptest.NewRequest(http.MethodGet, "/call-api", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid"})

	rec := httptest.NewRecorder()
	a.handleCallAPI(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without a cached token, got %d", rec.Code)
	}
}

func TestExchangeCodeSurfacesEndpointErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := newTestApp(srv.URL, "http://api.example.com")

	if _, err := a.exchangeCode(context.Background(), "code", "verifier"); err == nil {
		t.Fatal("expected an error on a 500 response")
	} else if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error should mention the status code, got %v", err)
	}
}

func TestCallProtectedAPIStatusFormatting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			http.Error(w, "boom", http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	a := newTestApp("http://token.example.com", srv.URL)

	t.Run("success", func(t *testing.T) {
		result, err := a.callProtectedAPI(context.Background(), "token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.StatusCode != http.StatusOK || !strings.Contains(result.Status, "Success") {
			t.Errorf("unexpected result: %+v", result)
		}
	})

	t.Run("marks non-2xx responses as failed", func(t *testing.T) {
		a := a
		a.apiURL = srv.URL + "/fail"

		result, err := a.callProtectedAPI(context.Background(), "token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.StatusCode != http.StatusForbidden || strings.Contains(result.Status, "Success") {
			t.Errorf("unexpected result: %+v", result)
		}
	})
}

func TestHandleLogoutClearsSession(t *testing.T) {
	a := newTestApp("http://token.example.com", "http://api.example.com")
	a.setSession("sid", session{AccessToken: "at"})

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid"})

	rec := httptest.NewRecorder()
	a.handleLogout(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}

	// セッション削除と cookie 失効（Max-Age=0）の両方を確認する。
	if _, ok := a.getSession("sid"); ok {
		t.Error("logout should delete the session")
	}
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, sessionCookieName+"=") || !strings.Contains(setCookie, "Max-Age=0") {
		t.Errorf("logout should expire the session cookie, got %q", setCookie)
	}
}

func parseSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			return c.Value
		}
	}
	t.Fatal("response did not set a session cookie")
	return ""
}

func mustGetSession(t *testing.T, a *app, id string) session {
	t.Helper()

	sess, ok := a.getSession(id)
	if !ok {
		t.Fatalf("session %q not found", id)
	}
	return sess
}

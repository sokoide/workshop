package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "oauth2_client_session"

type session struct {
	State        string
	CodeVerifier string
	AccessToken  string
	Expiry       time.Time
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
	Scope       string `json:"scope"`
}

type apiCallResult struct {
	StatusCode int
	Status     string
	Body       string
}

type app struct {
	clientID     string
	redirectURI  string
	issuerURL    string
	authURL      string
	tokenURL     string
	apiURL       string
	scope        string
	mu           sync.Mutex
	sessionStore map[string]session
}

func main() {
	a := &app{
		clientID:     getenv("OAUTH_CLIENT_ID", "workshop-client"),
		redirectURI:  getenv("OAUTH_REDIRECT_URI", "http://localhost:3000/callback"),
		issuerURL:    getenv("OAUTH_ISSUER_URL", "http://localhost:8080/realms/workshop"),
		authURL:      getenv("OAUTH_AUTH_URL", "http://localhost:8080/realms/workshop/protocol/openid-connect/auth"),
		tokenURL:     getenv("OAUTH_TOKEN_URL", "http://localhost:8080/realms/workshop/protocol/openid-connect/token"),
		apiURL:       getenv("RESOURCE_API_URL", "http://localhost:8081/api/profile"),
		scope:        getenv("OAUTH_SCOPE", "openid profile email"),
		sessionStore: make(map[string]session),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleHome)
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/callback", a.handleCallback)
	mux.HandleFunc("/call-api", a.handleCallAPI)
	mux.HandleFunc("/logout", a.handleLogout)

	server := &http.Server{
		Addr:              ":3000",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("oauth2 client app started on :3000")
	log.Fatal(server.ListenAndServe())
}

func (a *app) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	sessID := getOrSetSessionID(w, r)
	sess, _ := a.getSession(sessID)
	hasToken := strings.TrimSpace(sess.AccessToken) != ""

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html>
<head><meta charset="utf-8"><title>OAuth2 Client App</title></head>
<body>
<h1>OAuth2 Client App</h1>
<p><a href="/login">Login with Keycloak</a></p>
<p><a href="/call-api">Call Protected API</a></p>
<p><a href="/logout">Logout</a></p>
<ul>
  <li>issuer: %s</li>
  <li>client_id: %s</li>
  <li>redirect_uri: %s</li>
  <li>scope: %s</li>
  <li>token_cached: %t</li>
</ul>
</body>
</html>`, htmlEscape(a.issuerURL), htmlEscape(a.clientID), htmlEscape(a.redirectURI), htmlEscape(a.scope), hasToken)
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	_, _ = w.Write([]byte("OK"))
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	sessID := getOrSetSessionID(w, r)
	state, err := randomURLSafe(32)
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}
	verifier, err := randomURLSafe(48)
	if err != nil {
		http.Error(w, "Failed to generate code verifier", http.StatusInternalServerError)
		return
	}

	a.setSession(sessID, session{
		State:        state,
		CodeVerifier: verifier,
		Expiry:       time.Now().Add(10 * time.Minute),
	})

	values := url.Values{}
	values.Set("client_id", a.clientID)
	values.Set("redirect_uri", a.redirectURI)
	values.Set("response_type", "code")
	values.Set("scope", a.scope)
	values.Set("state", state)
	values.Set("code_challenge", pkceChallenge(verifier))
	values.Set("code_challenge_method", "S256")

	http.Redirect(w, r, a.authURL+"?"+values.Encode(), http.StatusFound)
}

func (a *app) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if errText := strings.TrimSpace(r.URL.Query().Get("error")); errText != "" {
		http.Error(w, "Authorization failed: "+errText, http.StatusBadRequest)
		return
	}

	sessID := getOrSetSessionID(w, r)
	sess, ok := a.getSession(sessID)
	if !ok {
		http.Error(w, "Session not found", http.StatusBadRequest)
		return
	}
	if time.Now().After(sess.Expiry) {
		http.Error(w, "Session expired", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != sess.State {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	tokenResp, err := a.exchangeCode(r.Context(), code, sess.CodeVerifier)
	if err != nil {
		log.Printf("token exchange failed: %v", err)
		http.Error(w, "Token exchange failed", http.StatusBadGateway)
		return
	}

	sess.AccessToken = tokenResp.AccessToken
	sess.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	a.setSession(sessID, sess)

	apiResult, err := a.callProtectedAPI(r.Context(), tokenResp.AccessToken)
	if err != nil {
		log.Printf("api call failed: %v", err)
		apiResult = apiCallResult{
			StatusCode: http.StatusBadGateway,
			Status:     "❌ API Call Failed (Network Error)",
			Body:       err.Error(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"message":      "login completed",
		"token_type":   tokenResp.TokenType,
		"scope":        tokenResp.Scope,
		"access_token": tokenResp.AccessToken,
		"id_token":     tokenResp.IDToken,
		"api_result":   apiResult,
	}); err != nil {
		log.Printf("callback response encode failed: %v", err)
	}
}

func (a *app) handleCallAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	sessID := getOrSetSessionID(w, r)
	sess, ok := a.getSession(sessID)
	if !ok || strings.TrimSpace(sess.AccessToken) == "" {
		http.Error(w, "Access token not found. Visit /login first.", http.StatusUnauthorized)
		return
	}

	result, err := a.callProtectedAPI(r.Context(), sess.AccessToken)
	if err != nil {
		log.Printf("api call failed: %v", err)
		http.Error(w, "Failed to call protected API", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("call api response encode failed: %v", err)
	}
}

func (a *app) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && strings.TrimSpace(cookie.Value) != "" {
		a.mu.Lock()
		delete(a.sessionStore, cookie.Value)
		a.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *app) exchangeCode(ctx context.Context, code, verifier string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", a.clientID)
	form.Set("code", code)
	form.Set("redirect_uri", a.redirectURI)
	form.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return tokenResponse{}, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return tokenResponse{}, err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return tokenResponse{}, fmt.Errorf("empty access_token")
	}
	return tokenResp, nil
}

func (a *app) callProtectedAPI(ctx context.Context, accessToken string) (apiCallResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.apiURL, nil)
	if err != nil {
		return apiCallResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiCallResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiCallResult{}, err
	}

	status := fmt.Sprintf("✅ API Call Success! (%s)", resp.Status)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		status = fmt.Sprintf("⚠️ API Call Failed (%s)", resp.Status)
	}

	return apiCallResult{
		StatusCode: resp.StatusCode,
		Status:     status,
		Body:       string(body),
	}, nil
}

func (a *app) getSession(id string) (session, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sess, ok := a.sessionStore[id]
	return sess, ok
}

func (a *app) setSession(id string, sess session) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.sessionStore[id] = sess
}

func getOrSetSessionID(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value
	}

	sessionID, err := randomURLSafe(32)
	if err != nil {
		log.Fatalf("failed to create session ID: %v", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return sessionID
}

func randomURLSafe(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

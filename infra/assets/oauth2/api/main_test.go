package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	handler := http.HandlerFunc(healthHandler)

	t.Run("returns OK on GET", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "OK" {
			t.Errorf("expected body OK, got %q", rec.Body.String())
		}
	})

	t.Run("rejects non-GET methods", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/health", nil))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rec.Code)
		}
	})
}

func TestProfileHandler(t *testing.T) {
	// keyfunc は Bearer トークンが来て初めて呼ばれる。ここでは
	// 認証前のガード（メソッドとヘッダ検証）だけを nil のまま検証する。
	handler := profileHandler(nil, "http://issuer.example.com", "workshop-client")

	t.Run("rejects non-GET methods", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/profile", nil))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("requires a Bearer token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/profile", nil))

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without Authorization header, got %d", rec.Code)
		}
	})

	t.Run("rejects an empty Bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
		req.Header.Set("Authorization", "Bearer ")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for an empty token, got %d", rec.Code)
		}
	})
}

func TestGetenv(t *testing.T) {
	t.Run("returns the environment value when set", func(t *testing.T) {
		t.Setenv("OIDC_ISSUER", "http://issuer.example.com")

		if got := getenv("OIDC_ISSUER", "fallback"); got != "http://issuer.example.com" {
			t.Errorf("expected the env value, got %q", got)
		}
	})

	t.Run("falls back when unset or blank", func(t *testing.T) {
		t.Setenv("OIDC_ISSUER", "   ")

		if got := getenv("OIDC_ISSUER", "fallback"); got != "fallback" {
			t.Errorf("expected the fallback for a blank value, got %q", got)
		}
		if got := getenv("OIDC_UNSET_VAR", "fallback"); got != "fallback" {
			t.Errorf("expected the fallback, got %q", got)
		}
	})
}

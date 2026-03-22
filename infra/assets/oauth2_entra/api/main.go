package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	TenantID        string
	APIClientID     string
	RequiredScope   string
	RequiredAppRole string
	Port            string
}

func loadConfig() (*Config, error) {
	c := &Config{
		TenantID:        os.Getenv("TENANT_ID"),
		APIClientID:     os.Getenv("API_CLIENT_ID"),
		RequiredScope:   os.Getenv("REQUIRED_SCOPE"),
		RequiredAppRole: os.Getenv("REQUIRED_APP_ROLE"),
		Port:            os.Getenv("PORT"),
	}

	if c.TenantID == "" || c.APIClientID == "" {
		return nil, fmt.Errorf("TENANT_ID and API_CLIENT_ID must be set")
	}
	if c.RequiredScope == "" {
		c.RequiredScope = "access_as_user"
	}
	if c.RequiredAppRole == "" {
		c.RequiredAppRole = "Svc.Invoke"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	return c, nil
}

func main() {
	// Use structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	jwksURL := fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", cfg.TenantID)
	expectedIssuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", cfg.TenantID)

	// Fetch JWKS for token validation
	kf, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		slog.Error("failed to create keyfunc", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/profile", profileHandler(cfg, kf.Keyfunc, expectedIssuer))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	slog.Info("Starting API server", "port", cfg.Port, "client_id", cfg.APIClientID)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func profileHandler(cfg *Config, kf jwt.Keyfunc, expectedIssuer string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			slog.Warn("missing or invalid authorization header", "path", r.URL.Path)
			http.Error(w, "Unauthorized: missing Bearer token", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate JWT signature, issuer, and audience
		token, err := jwt.Parse(tokenStr, kf,
			jwt.WithIssuer(expectedIssuer),
			jwt.WithAudience(cfg.APIClientID),
			jwt.WithValidMethods([]string{"RS256"}),
		)
		if err != nil {
			slog.Warn("token validation failed", "error", err)
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			slog.Warn("invalid token claims", "valid", token.Valid)
			http.Error(w, "Unauthorized: invalid claims", http.StatusUnauthorized)
			return
		}

		// Check for App Roles (M2M flow)
		hasValidRole := checkRole(claims, cfg.RequiredAppRole)
		// Check for Scopes (User-delegated flow)
		hasValidScope := checkScope(claims, cfg.RequiredScope)

		if !hasValidScope && !hasValidRole {
			slog.Warn("insufficient permissions", "user", claims["name"], "roles", claims["roles"], "scp", claims["scp"])
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"user":   claims["name"],
			"claims": claims,
		})
	}
}

func checkRole(claims jwt.MapClaims, requiredRole string) bool {
	rawRoles, ok := claims["roles"].([]any)
	if !ok {
		return false
	}
	for _, r := range rawRoles {
		if role, ok := r.(string); ok && role == requiredRole {
			return true
		}
	}
	return false
}

func checkScope(claims jwt.MapClaims, requiredScope string) bool {
	scp, ok := claims["scp"].(string)
	if !ok {
		return false
	}
	for _, s := range strings.Fields(scp) {
		if s == requiredScope {
			return true
		}
	}
	return false
}

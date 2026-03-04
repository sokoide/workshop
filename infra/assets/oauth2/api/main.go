package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type profileResponse struct {
	Message string `json:"message"`
	Sub     string `json:"sub"`
}

func main() {
	port := getenv("PORT", "8081")
	issuer := getenv("OIDC_ISSUER", "http://localhost:8080/realms/workshop")
	audience := getenv("OIDC_AUDIENCE", "workshop-client")
	jwksURL := getenv("OIDC_JWKS_URL", issuer+"/protocol/openid-connect/certs")

	kf, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		log.Fatalf("failed to create keyfunc: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/profile", profileHandler(kf, issuer, audience))

	addr := ":" + port
	log.Printf("resource server started on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	_, _ = w.Write([]byte("OK"))
}

func profileHandler(kf keyfunc.Keyfunc, issuer, audience string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			unauthorized(w)
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenStr == "" {
			unauthorized(w)
			return
		}

		token, err := jwt.Parse(
			tokenStr,
			kf.Keyfunc,
			jwt.WithAudience(audience),
			jwt.WithIssuer(issuer),
			jwt.WithValidMethods([]string{"RS256"}),
		)
		if err != nil || !token.Valid {
			log.Printf("token validation failed: %v", err)
			unauthorized(w)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			unauthorized(w)
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			unauthorized(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(profileResponse{
			Message: "Hello, authenticated SWE!",
			Sub:     sub,
		}); err != nil {
			log.Printf("response encode failed: %v", err)
		}
	}
}

func unauthorized(w http.ResponseWriter) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

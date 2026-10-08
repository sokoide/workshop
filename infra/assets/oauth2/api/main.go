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

		// Parse with signature, issuer, and algorithm checks. The audience
		// check is done below instead of via jwt.WithAudience: recent
		// Keycloak versions may issue access tokens whose aud is only
		// "account" even with an Audience mapper configured (see the
		// workshop doc). In that case azp ("authorized party") still
		// reliably names the client the token was issued to.
		token, err := jwt.Parse(
			tokenStr,
			kf.Keyfunc,
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

		if !tokenIntendedFor(claims, audience) {
			log.Printf("token is not intended for audience %q", audience)
			unauthorized(w)
			return
		}
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

// tokenIntendedFor reports whether the token was issued for the expected
// audience: the standard path is aud containing the client ID, with azp as
// the fallback for providers whose access tokens carry only aud="account".
func tokenIntendedFor(claims jwt.MapClaims, audience string) bool {
	if aud, ok := claims["aud"].(string); ok && aud == audience {
		return true
	}
	if auds, ok := claims["aud"].([]any); ok {
		for _, a := range auds {
			if s, ok := a.(string); ok && s == audience {
				return true
			}
		}
	}
	azp, ok := claims["azp"].(string)
	return ok && azp == audience
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

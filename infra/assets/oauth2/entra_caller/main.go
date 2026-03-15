package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type managedIdentityTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresOn   string `json:"expires_on"`
	Resource    string `json:"resource"`
	TokenType   string `json:"token_type"`
}

type proxyResponse struct {
	TargetURL   string `json:"target_url"`
	StatusCode  int    `json:"status_code"`
	Body        string `json:"body"`
	TokenSource string `json:"token_source"`
}

func main() {
	port := getenv("PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/call-api", callAPIHandler)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("caller app started on :%s", port)
	log.Fatal(server.ListenAndServe())
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	_, _ = w.Write([]byte("OK"))
}

func callAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	targetURL := getenv("TARGET_API_URL", "")
	resource := getenv("TARGET_API_RESOURCE", "")
	if targetURL == "" || resource == "" {
		http.Error(w, "TARGET_API_URL and TARGET_API_RESOURCE are required", http.StatusInternalServerError)
		return
	}

	token, err := fetchManagedIdentityToken(resource, getenv("MANAGED_IDENTITY_CLIENT_ID", ""))
	if err != nil {
		log.Printf("managed identity token fetch failed: %v", err)
		http.Error(w, "Failed to fetch managed identity token", http.StatusBadGateway)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create downstream request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("downstream API call failed: %v", err)
		http.Error(w, "Failed to call downstream API", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("downstream response read failed: %v", err)
		http.Error(w, "Failed to read downstream response", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(proxyResponse{
		TargetURL:   targetURL,
		StatusCode:  resp.StatusCode,
		Body:        string(body),
		TokenSource: "managed-identity",
	}); err != nil {
		log.Printf("proxy response encode failed: %v", err)
	}
}

func fetchManagedIdentityToken(resource, clientID string) (string, error) {
	identityEndpoint := getenv("IDENTITY_ENDPOINT", "")
	identityHeader := getenv("IDENTITY_HEADER", "")
	if identityEndpoint == "" || identityHeader == "" {
		return "", fmt.Errorf("IDENTITY_ENDPOINT and IDENTITY_HEADER must be set by App Service managed identity")
	}

	params := url.Values{}
	params.Set("resource", resource)
	params.Set("api-version", "2019-08-01")
	if clientID != "" {
		params.Set("client_id", clientID)
	}

	req, err := http.NewRequest(http.MethodGet, identityEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-IDENTITY-HEADER", identityHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("identity endpoint returned status %d", resp.StatusCode)
	}

	var tokenResp managedIdentityTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", fmt.Errorf("identity endpoint returned empty access token")
	}

	return tokenResp.AccessToken, nil
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

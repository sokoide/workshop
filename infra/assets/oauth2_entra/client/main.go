package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type App struct {
	httpClient *http.Client
	cred       *azidentity.DefaultAzureCredential
}

func main() {
	// Use structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create credential once at startup (reuse across requests)
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		slog.Error("failed to create credential", "error", err)
		os.Exit(1)
	}

	app := &App{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		cred: cred,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handler)

	slog.Info("Starting client server", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func (app *App) handler(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context = r.Context()

	// 2. Attempt to acquire a token
	scope := os.Getenv("API_SCOPE")
	if scope == "" {
		slog.Warn("API_SCOPE is not set, using default Graph scope")
		scope = "https://graph.microsoft.com/.default"
	}

	token, err := app.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
	})

	if err != nil {
		slog.Error("token acquisition failed", "error", err, "scope", scope)
		http.Error(w, fmt.Sprintf("❌ Token Error: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Display success message (with safety check)
	tokenStr := token.Token
	if len(tokenStr) < 10 {
		slog.Error("invalid token length", "length", len(tokenStr))
		http.Error(w, fmt.Sprintf("❌ Invalid token: too short (%d chars)", len(tokenStr)),
			http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "✅ Managed Identity Success!\n")
	fmt.Fprintf(w, "Token (first 10 chars): %s...\n", tokenStr[:10])
	fmt.Fprintf(w, "Expires On: %v\n\n", token.ExpiresOn)
	slog.Info("token retrieved successfully", "scope", scope, "expires", token.ExpiresOn)

	// 4. Call the API Endpoint
	apiEndpoint := os.Getenv("API_ENDPOINT")
	if apiEndpoint == "" {
		fmt.Fprintln(w, "⚠️ API_ENDPOINT is not set. Skipping API call.")
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiEndpoint, nil)
	if err != nil {
		slog.Error("failed to create API request", "error", err, "endpoint", apiEndpoint)
		fmt.Fprintf(w, "❌ Failed to create API request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, err := app.httpClient.Do(req)
	if err != nil {
		slog.Error("API call failed", "error", err, "endpoint", apiEndpoint)
		fmt.Fprintf(w, "❌ API Call Failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read response body", "error", err)
		fmt.Fprintf(w, "⚠️ API call completed but failed to read response: %v\n", err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Fprintf(w, "✅ API Call Success! (Status: %s)\n", resp.Status)
	} else {
		fmt.Fprintf(w, "⚠️ API Call Failed (Status: %s)\n", resp.Status)
	}
	fmt.Fprintf(w, "Response Body:\n%s\n", string(body))
	slog.Info("API call completed", "status", resp.Status, "endpoint", apiEndpoint)
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func main() {
	// Get port from environment variable (App Service default is 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Authenticate using Managed Identity
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			http.Error(w, "Failed to create credential: "+err.Error(), 500)
			return
		}

		// 2. Attempt to acquire a token
		scope := os.Getenv("API_SCOPE")
		if scope == "" {
			fmt.Fprintln(w, "Warning: API_SCOPE is not set. Using default...")
			scope = "https://graph.microsoft.com/.default" // For testing
		}

		token, err := cred.GetToken(context.Background(), policy.TokenRequestOptions{
			Scopes: []string{scope},
		})

		if err != nil {
			// Display error in browser on failure
			fmt.Fprintf(w, "❌ Token Error: %v\n", err)
			log.Printf("Token Error: %v", err)
			return
		}

		// 3. Display success message in browser
		fmt.Fprintf(w, "✅ Managed Identity Success!\n")
		fmt.Fprintf(w, "Token (first 10 chars): %s...\n", token.Token[:10])
		fmt.Fprintf(w, "Expires On: %v\n", token.ExpiresOn)
		
		log.Printf("Successfully retrieved token for scope: %s", scope)
	})

	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

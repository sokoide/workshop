package main

import (
	"fmt"
	"net/http"
	"workshop/greeting/presentation"
	"workshop/greeting/infra"
	"workshop/greeting/usecase"
)

func main() {
	// 1. Dependency Injection (Composition Root)
	repo := infra.NewMemoryUserRepo()
	uc := &usecase.GreetingUseCase{Repo: repo}
	handler := &presentation.GreetingHandler{UC: uc}

	// 2. Start Server
	port := ":8080"
	fmt.Printf("Starting server on http://localhost%s (try ?id=1)\n", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

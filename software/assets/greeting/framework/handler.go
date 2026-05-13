package framework

import (
	"net/http"
	"workshop/greeting/usecase"
)

type GreetingHandler struct {
	UC *usecase.GreetingUseCase
}

func (h *GreetingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	msg, err := h.UC.Execute(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Write([]byte(msg))
}

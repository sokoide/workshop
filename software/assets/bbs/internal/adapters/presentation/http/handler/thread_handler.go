package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/usecase"
)

type ThreadHandler struct {
	listThreads  usecase.ListThreadsInputPort
	createThread usecase.CreateThreadInputPort
}

func NewThreadHandler(listThreads usecase.ListThreadsInputPort, createThread usecase.CreateThreadInputPort) *ThreadHandler {
	return &ThreadHandler{listThreads: listThreads, createThread: createThread}
}

func (h *ThreadHandler) ListThreads(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "board name is required")
		return
	}

	out, err := h.listThreads.Execute(r.Context(), usecase.ListThreadsInput{BoardName: name})
	if err != nil {
		if errors.Is(err, domain.ErrBoardNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type createThreadRequest struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	OwnerOnly bool   `json:"owner_only"`
}

func (h *ThreadHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "board name is required")
		return
	}

	var req createThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
		BoardName: name,
		Title:     req.Title,
		Author:    req.Author,
		Body:      req.Body,
		OwnerOnly: req.OwnerOnly,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBoardNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrEmptyTitle), errors.Is(err, domain.ErrEmptyBody):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

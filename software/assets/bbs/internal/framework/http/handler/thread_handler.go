package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/usecase"
)

type ThreadHandler struct {
	listThreads  *usecase.ListThreadsUseCase
	createThread *usecase.CreateThreadUseCase
}

func NewThreadHandler(listThreads *usecase.ListThreadsUseCase, createThread *usecase.CreateThreadUseCase) *ThreadHandler {
	return &ThreadHandler{listThreads: listThreads, createThread: createThread}
}

func (h *ThreadHandler) ListThreads(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "board slug is required")
		return
	}

	out, err := h.listThreads.Execute(r.Context(), usecase.ListThreadsInput{BoardSlug: slug})
	if err != nil {
		if errors.Is(err, domain.ErrBoardNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type createThreadRequest struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Body   string `json:"body"`
}

func (h *ThreadHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "board slug is required")
		return
	}

	var req createThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
		BoardSlug: slug,
		Title:     req.Title,
		Author:    req.Author,
		Body:      req.Body,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBoardNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrEmptyTitle), errors.Is(err, domain.ErrEmptyBody):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

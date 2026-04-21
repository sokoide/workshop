package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/usecase"
)

type PostHandler struct {
	listPosts  *usecase.ListPostsUseCase
	createPost *usecase.CreatePostUseCase
}

func NewPostHandler(listPosts *usecase.ListPostsUseCase, createPost *usecase.CreatePostUseCase) *PostHandler {
	return &PostHandler{listPosts: listPosts, createPost: createPost}
}

func (h *PostHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	threadID, err := strconv.ParseInt(r.PathValue("threadID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid thread id")
		return
	}

	out, err := h.listPosts.Execute(r.Context(), usecase.ListPostsInput{ThreadID: threadID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type createPostRequest struct {
	Author string `json:"author"`
	Body   string `json:"body"`
	Sage   bool   `json:"sage"`
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	threadID, err := strconv.ParseInt(r.PathValue("threadID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid thread id")
		return
	}

	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	out, err := h.createPost.Execute(r.Context(), usecase.CreatePostInput{
		ThreadID: threadID,
		Author:   req.Author,
		Body:     req.Body,
		Sage:     req.Sage,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrThreadNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrEmptyBody):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

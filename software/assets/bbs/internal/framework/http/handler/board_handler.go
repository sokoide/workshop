package handler

import (
	"net/http"

	"github.com/sokoide/cleanarch1/internal/usecase"
)

type BoardHandler struct {
	listBoards *usecase.ListBoardsUseCase
}

func NewBoardHandler(listBoards *usecase.ListBoardsUseCase) *BoardHandler {
	return &BoardHandler{listBoards: listBoards}
}

func (h *BoardHandler) ListBoards(w http.ResponseWriter, r *http.Request) {
	out, err := h.listBoards.Execute(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

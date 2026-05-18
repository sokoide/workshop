package handler

import (
	"net/http"

	"github.com/sokoide/cleanarch1/internal/usecase"
)

type BoardHandler struct {
	listBoards usecase.ListBoardsInputPort
}

func NewBoardHandler(listBoards usecase.ListBoardsInputPort) *BoardHandler {
	return &BoardHandler{listBoards: listBoards}
}

func (h *BoardHandler) ListBoards(w http.ResponseWriter, r *http.Request) {
	out, err := h.listBoards.Execute(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

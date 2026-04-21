package http

import (
	"net/http"

	"github.com/sokoide/cleanarch1/internal/framework/http/handler"
	"github.com/sokoide/cleanarch1/internal/framework/http/middleware"
)

func NewRouter(
	boardHandler *handler.BoardHandler,
	threadHandler *handler.ThreadHandler,
	postHandler *handler.PostHandler,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/boards", boardHandler.ListBoards)
	mux.HandleFunc("GET /api/boards/{slug}/threads", threadHandler.ListThreads)
	mux.HandleFunc("POST /api/boards/{slug}/threads", threadHandler.CreateThread)
	mux.HandleFunc("GET /api/threads/{threadID}/posts", postHandler.ListPosts)
	mux.HandleFunc("POST /api/threads/{threadID}/posts", postHandler.CreatePost)

	return middleware.Logging(mux)
}

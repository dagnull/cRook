package web

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dagnull/cRook/internal/room"
	"github.com/dagnull/cRook/internal/session"
)

func NewRouter(mgr *room.Manager, sessions *session.Manager, uiFS fs.FS) http.Handler {
	InitTemplates(uiFS)

	r := chi.NewRouter()
	r.Use(recoverMiddleware)
	r.Use(loggingMiddleware)
	r.Use(sessions.Middleware)

	h := &handlers{mgr: mgr, sessions: sessions}

	r.Get("/", h.home)
	r.Post("/session", h.setSession)

	r.Post("/rooms", h.createRoom)
	r.Get("/rooms/{id}", h.roomPage)
	r.Get("/rooms/{id}/ws", h.websocket)

	r.Handle("/static/*", http.StripPrefix("/static/", staticHandler(uiFS)))

	return r
}

package api

import (
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Router chi.Router
	Hub    *hub.Hub
}

func NewHandler(hub *hub.Hub) *Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/up"))

	return &Handler{
		Router: r,
		Hub:    hub,
	}
}

func (h *Handler) Routes() {
	ws := NewWSHandler(h.Hub)

	h.Router.Get("/ws", ws.Handle)
}

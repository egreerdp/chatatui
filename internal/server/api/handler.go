package api

import (
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

func (h *Handler) Routes() chi.Router {
	ws := NewWSHandler(h.Hub)

	h.Router.Get("/ws", ws.Handle)

	return h.Router
}

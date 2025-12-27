package api

import (
	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	Router chi.Router
	Hub    *hub.Hub
	DB     *repository.SQLiteDB
}

func NewHandler(hub *hub.Hub, db *repository.SQLiteDB) *Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/up"))

	return &Handler{
		Router: r,
		Hub:    hub,
		DB:     db,
	}
}

func (h *Handler) Routes() chi.Router {
	ws := NewWSHandler(h.Hub, h.DB)

	h.Router.Get("/ws/{roomID}", ws.Handle)

	return h.Router
}

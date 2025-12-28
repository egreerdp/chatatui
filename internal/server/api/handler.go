package api

import (
	"github.com/egreerdp/chatatui/internal/middleware"
	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	Router chi.Router
	Hub    *hub.Hub
	DB     *repository.SQLiteDB
}

func NewHandler(hub *hub.Hub, db *repository.SQLiteDB) *Handler {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Heartbeat("/up"))

	return &Handler{
		Router: r,
		Hub:    hub,
		DB:     db,
	}
}

func (h *Handler) Routes() chi.Router {
	ws := NewWSHandler(h.Hub, h.DB)
	register := NewRegisterHandler(h.DB)
	rooms := NewRoomsHandler(h.DB)

	h.Router.Post("/register", register.Handle)

	h.Router.Group(func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(h.DB.Users()))
		r.Get("/rooms", rooms.List)
		r.Get("/ws/{roomID}", ws.Handle)
	})

	return h.Router
}

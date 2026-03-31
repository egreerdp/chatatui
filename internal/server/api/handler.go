package api

import (
	"time"

	"github.com/egreerdp/chatatui/internal/config"
	"github.com/egreerdp/chatatui/internal/middleware"
	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/egreerdp/chatatui/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type ChatService interface {
	GetRoom(id uuid.UUID) (*service.RoomInfo, error)
	AddRoomMember(roomID, userID uuid.UUID) error
	GetMessageHistory(roomID uuid.UUID, limit, offset int) ([]service.MessageInfo, error)
	PersistMessage(content []byte, senderID, roomID uuid.UUID) (uuid.UUID, time.Time, error)
}

type Handler struct {
	Router      chi.Router
	Hub         *hub.Hub
	DB          *repository.PostgresDB
	ChatService ChatService
	Config      config.ServerConfig
	RateLimiter *middleware.RateLimiter
}

func NewHandler(hub *hub.Hub, db *repository.PostgresDB, cfg config.ServerConfig, rl *middleware.RateLimiter) *Handler {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Heartbeat("/up"))

	svc := service.NewChatService(db.Rooms(), db.Messages())

	return &Handler{
		Router:      r,
		Hub:         hub,
		DB:          db,
		ChatService: svc,
		Config:      cfg,
		RateLimiter: rl,
	}
}

func (h *Handler) Routes() chi.Router {
	ws := NewWSHandler(h.Hub, h.ChatService, h.Config.MessageHistoryLimit)
	register := NewRegisterHandler(h.DB)
	rooms := NewRoomsHandler(h.DB, h.Config.RoomListLimit)

	h.Router.Post("/register", register.Handle)

	h.Router.Group(func(r chi.Router) {
		r.Use(
			middleware.APIKeyAuth(h.DB.Users()),
			h.RateLimiter.Middleware,
		)

		r.Get("/rooms", rooms.List)
		r.Post("/rooms", rooms.Create)
		r.Get("/ws/{roomID}", ws.Handle)
	})

	return h.Router
}

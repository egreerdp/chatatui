package api

import (
	"time"

	"github.com/EwanGreer/chatatui/internal/config"
	"github.com/EwanGreer/chatatui/internal/middleware"
	"github.com/EwanGreer/chatatui/internal/server/hub"
	"github.com/EwanGreer/chatatui/internal/service"
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
	Router          chi.Router
	Hub             *hub.Hub
	ChatService     ChatService
	Config          config.ServerConfig
	RateLimiter     *middleware.RateLimiter
	userLookup      middleware.UserLookup
	wsHandler       *WSHandler
	registerHandler *RegisterHandler
	roomsHandler    *RoomsHandler
}

func NewHandler(h *hub.Hub, users middleware.UserLookup, userStore UserStore, roomStore RoomStore, svc ChatService, cfg config.ServerConfig, rl *middleware.RateLimiter) *Handler {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Heartbeat("/up"))

	return &Handler{
		Router:          r,
		Hub:             h,
		ChatService:     svc,
		Config:          cfg,
		RateLimiter:     rl,
		userLookup:      users,
		wsHandler:       NewWSHandler(h, svc, cfg.MessageHistoryLimit),
		registerHandler: NewRegisterHandler(userStore),
		roomsHandler:    NewRoomsHandler(roomStore, cfg.RoomListLimit),
	}
}

func (h *Handler) Routes() chi.Router {
	h.Router.Post("/register", h.registerHandler.Handle)

	h.Router.Group(func(r chi.Router) {
		r.Use(
			middleware.APIKeyAuth(h.userLookup),
			h.RateLimiter.Middleware,
		)

		r.Get("/rooms", h.roomsHandler.List)
		r.Post("/rooms", h.roomsHandler.Create)
		r.Get("/ws/{roomID}", h.wsHandler.Handle)
	})

	return h.Router
}

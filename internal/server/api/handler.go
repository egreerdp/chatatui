package api

import (
	"github.com/EwanGreer/chatatui/internal/config"
	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/EwanGreer/chatatui/internal/middleware"
	"github.com/EwanGreer/chatatui/internal/server/hub"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type ChatService interface {
	GetRoom(id uuid.UUID) (*domain.Room, error)
	JoinRoom(roomID, userID uuid.UUID) ([]domain.WireMessage, error)
	PublishMessage(content []byte, senderID uuid.UUID, senderName string, roomID uuid.UUID) (*domain.WireMessage, error)
}

type RoomHub interface {
	GetOrCreateSession(uuid.UUID) (*hub.Session, error)
	ActiveCount() int
}

type Handler struct {
	Router          chi.Router
	ChatService     ChatService
	Config          config.ServerConfig
	RateLimiter     *middleware.RateLimiter
	userLookup      middleware.UserLookup
	wsHandler       *WSHandler
	registerHandler *RegisterHandler
	roomsHandler    *RoomsHandler
}

func NewHandler(h RoomHub, users middleware.UserLookup, userStore UserStore, roomStore RoomStore, svc ChatService, cfg config.ServerConfig, rl *middleware.RateLimiter) *Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Heartbeat("/up"))

	return &Handler{
		Router:          r,
		ChatService:     svc,
		Config:          cfg,
		RateLimiter:     rl,
		userLookup:      users,
		wsHandler:       NewWSHandler(h, svc),
		registerHandler: NewRegisterHandler(userStore),
		roomsHandler:    NewRoomsHandler(roomStore, cfg.RoomListLimit),
	}
}

func (h *Handler) Routes() chi.Router {
	h.Router.Post("/register", h.registerHandler.Handle)

	h.Router.Group(func(r chi.Router) {
		r.Use( // TODO: standardise how these are passed
			middleware.APIKeyAuth(h.userLookup),
			h.RateLimiter.Middleware,
		)

		r.Get("/rooms", h.roomsHandler.Index)
		r.Post("/rooms", h.roomsHandler.Create)
		r.Get("/rooms/{id}", h.roomsHandler.Show)
		r.Get("/ws/{roomID}", h.wsHandler.Handle)
	})

	return h.Router
}

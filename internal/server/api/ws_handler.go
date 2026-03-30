package api

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/egreerdp/chatatui/internal/middleware"
	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WSHandler struct {
	hub                 *hub.Hub
	db                  *repository.PostgresDB
	messageHistoryLimit int
}

func NewWSHandler(h *hub.Hub, db *repository.PostgresDB, messageHistoryLimit int) *WSHandler {
	go func() {
		for {
			time.Sleep(time.Second * 5)
			slog.Debug("hub status", "room_count", len(h.Rooms))
		}
	}()

	return &WSHandler{
		hub:                 h,
		db:                  db,
		messageHistoryLimit: messageHistoryLimit,
	}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		http.Error(w, "room required", http.StatusBadRequest)
		return
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	dbRoom, err := h.db.Rooms().GetByID(roomUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get room", http.StatusInternalServerError)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		slog.Error("failed to accept websocket", "error", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	room, err := h.hub.GetOrCreateRoom(roomUUID)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "failed to join room")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if err := h.db.Rooms().AddMember(dbRoom.ID, user.ID); err != nil {
		slog.Error("failed to add room member", "error", err, "room_id", dbRoom.ID, "user_id", user.ID)
	}

	client := hub.NewClient(conn, user.ID, roomUUID, user.Name)
	room.Add(client)
	defer room.Remove(client)

	h.sendHistory(client, dbRoom.ID)

	client.Run(room, h.db.Messages())
}

func (h *WSHandler) sendHistory(client *hub.Client, roomID uuid.UUID) {
	messages, err := h.db.Messages().GetByRoom(roomID, h.messageHistoryLimit, 0)
	if err != nil {
		slog.Error("failed to get message history", "error", err, "room_id", roomID)
		return
	}

	// Send messages in chronological order (oldest first)
	for i := len(messages) - 1; i >= 0; i-- {
		wire := &hub.WireMessage{
			Type:      hub.MessageTypeChat,
			ID:        messages[i].ID.String(),
			Author:    messages[i].Sender.Name,
			Content:   string(messages[i].Content),
			Timestamp: messages[i].CreatedAt,
		}
		wireBytes, err := wire.Marshal()
		if err != nil {
			slog.Error("failed to marshal history message", "error", err, "room_id", roomID)
			continue
		}
		client.SendRaw(wireBytes)
	}
}

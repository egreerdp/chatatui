package api

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/EwanGreer/chatatui/internal/middleware"
	"github.com/EwanGreer/chatatui/internal/server/hub"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WSHandler struct {
	hub                 RoomHub
	svc                 ChatService
	messageHistoryLimit int
}

func NewWSHandler(h RoomHub, svc ChatService, messageHistoryLimit int) *WSHandler {
	go func() {
		for {
			time.Sleep(time.Second * 5)
			slog.Debug("hub status", "room_count", h.ActiveCount())
		}
	}()

	return &WSHandler{
		hub:                 h,
		svc:                 svc,
		messageHistoryLimit: messageHistoryLimit,
	}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		writeError(w, http.StatusBadRequest, "ROOM_REQUIRED", "room required")
		return
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ROOM_ID", "invalid room id")
		return
	}

	roomInfo, err := h.svc.GetRoom(roomUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "ROOM_NOT_FOUND", "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get room")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		slog.Error("failed to accept websocket", "error", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	session, err := h.hub.GetOrCreateSession(roomUUID)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "failed to join room")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if err := h.svc.AddRoomMember(roomInfo.ID, user.ID); err != nil {
		slog.Error("failed to add room member", "error", err, "room_id", roomInfo.ID, "user_id", user.ID)
	}

	client := hub.NewClient(conn, user.ID, roomUUID, user.Name)
	session.AddClient(client)
	defer session.RemoveClient(client)

	h.sendHistory(client, roomInfo.ID)

	client.Run(session, h.svc) // blocking
}

func (h *WSHandler) sendHistory(client *hub.Client, roomID uuid.UUID) {
	messages, err := h.svc.GetMessageHistory(roomID, h.messageHistoryLimit, 0)
	if err != nil {
		slog.Error("failed to get message history", "error", err, "room_id", roomID)
		return
	}

	// Send messages in chronological order (oldest first)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := &hub.Message{
			Type:      hub.MessageTypeChat,
			ID:        messages[i].ID.String(),
			Author:    messages[i].Author,
			Content:   messages[i].Content,
			Timestamp: messages[i].CreatedAt,
		}
		wireBytes, err := msg.Marshal()
		if err != nil {
			slog.Error("failed to marshal history message", "error", err, "room_id", roomID)
			continue
		}
		client.SendRaw(wireBytes)
	}
}

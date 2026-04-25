package api

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/EwanGreer/chatatui/internal/middleware"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type WSHandler struct {
	hub RoomHub
	svc ChatService
}

func NewWSHandler(h RoomHub, svc ChatService) *WSHandler {
	go func() {
		for {
			time.Sleep(time.Second * 5)
			slog.Debug("hub status", "room_count", h.ActiveCount())
		}
	}()

	return &WSHandler{hub: h, svc: svc}
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
		if errors.Is(err, domain.ErrNotFound) {
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
	newRoomConn(conn, session, roomInfo.ID, user, h.svc).serve()
}

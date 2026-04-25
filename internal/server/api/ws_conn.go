package api

import (
	"log/slog"

	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/EwanGreer/chatatui/internal/server/hub"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type roomConn struct {
	client  *hub.Client
	session *hub.Session
	user    *domain.User
	roomID  uuid.UUID
	svc     ChatService
}

func newRoomConn(conn *websocket.Conn, session *hub.Session, roomID uuid.UUID, user *domain.User, svc ChatService) *roomConn {
	return &roomConn{
		client:  hub.NewClient(conn, user.ID, roomID, user.Name),
		session: session,
		user:    user,
		roomID:  roomID,
		svc:     svc,
	}
}

func (rc *roomConn) serve() {
	history, err := rc.svc.JoinRoom(rc.roomID, rc.user.ID)
	if err != nil {
		slog.Error("failed to join room", "error", err, "room_id", rc.roomID, "user_id", rc.user.ID)
	}

	rc.session.AddClient(rc.client)
	defer rc.session.RemoveClient(rc.client)

	rc.sendHistory(history)

	for raw := range rc.client.Run(rc.session) {
		rc.handleMessage(raw)
	}
}

func (rc *roomConn) handleMessage(raw []byte) {
	wire, err := rc.svc.PublishMessage(raw, rc.user.ID, rc.user.Name, rc.roomID)
	if err != nil {
		slog.Error("failed to publish message", "error", err, "room_id", rc.roomID, "user_id", rc.user.ID)
		rc.client.SendError("could not send message")
		return
	}

	wireBytes, err := wire.Marshal()
	if err != nil {
		slog.Error("failed to marshal message", "error", err, "room_id", rc.roomID, "user_id", rc.user.ID)
		return
	}

	rc.session.Broadcast(wireBytes, rc.client)
}

func (rc *roomConn) sendHistory(messages []domain.WireMessage) {
	for _, m := range messages {
		wireBytes, err := m.Marshal()
		if err != nil {
			slog.Error("failed to marshal history message", "error", err, "message_id", m.ID)
			continue
		}
		rc.client.SendRaw(wireBytes)
	}
}

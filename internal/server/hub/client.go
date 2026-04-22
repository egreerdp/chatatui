package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/EwanGreer/chatatui/internal/limits"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// MessagePersister abstracts message persistence so the hub package
// does not depend on the repository layer.
type MessagePersister interface {
	PersistMessage(content []byte, senderID, roomID uuid.UUID) (id uuid.UUID, createdAt time.Time, err error)
}

type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	UserID   uuid.UUID
	RoomID   uuid.UUID
	Username string
}

func NewClient(conn *websocket.Conn, userID, roomID uuid.UUID, username string) *Client {
	return &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		UserID:   userID,
		RoomID:   roomID,
		Username: username,
	}
}

func (c *Client) Run(session *Session, persister MessagePersister) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.writePump(ctx)
	c.readPump(ctx, session, persister) // blocking
}

func (c *Client) readPump(ctx context.Context, session *Session, persister MessagePersister) {
	defer func() { _ = c.conn.CloseNow() }()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}

		if len(data) > limits.MaxMessageLength {
			errMsg := &Message{
				Type:      MessageTypeError,
				Content:   fmt.Sprintf("message too long (max %d characters)", limits.MaxMessageLength),
				Timestamp: time.Now(),
			}
			if errBytes, err := errMsg.Marshal(); err == nil {
				c.Send(errBytes)
			}
			continue
		}

		var peek Message
		if json.Unmarshal(data, &peek) == nil && peek.Type == MessageTypeTyping {
			typingMsg := &Message{
				Type:      MessageTypeTyping,
				Author:    c.Username,
				Timestamp: time.Now(),
			}
			typingBytes, err := typingMsg.Marshal()
			if err != nil {
				slog.Error("failed to marshal typing event", "error", err, "user_id", c.UserID)
				continue
			}
			session.Broadcast(typingBytes, c)
			continue
		}

		msgID, createdAt, err := persister.PersistMessage(data, c.UserID, c.RoomID)
		if err != nil {
			slog.Error("failed to persist message", "error", err, "room_id", c.RoomID, "user_id", c.UserID)
		}

		wire := &Message{
			Type:    MessageTypeChat,
			ID:      msgID.String(),
			Author:  c.Username,
			Content: string(data),
		}
		if createdAt.IsZero() {
			wire.Timestamp = time.Now()
		} else {
			wire.Timestamp = createdAt
		}

		wireBytes, err := wire.Marshal()
		if err != nil {
			slog.Error("failed to marshal message", "error", err, "room_id", c.RoomID, "user_id", c.UserID)
			continue
		}

		session.Broadcast(wireBytes, c)
	}
}

func (c *Client) writePump(ctx context.Context) {
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			if err := c.conn.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
		slog.Debug("message sent to client", "user_id", c.UserID, "room_id", c.RoomID)
	default:
	}
}

func (c *Client) SendRaw(msg []byte) {
	select {
	case c.send <- msg:
	default:
		slog.Warn("client send buffer full, dropping message", "user_id", c.UserID, "room_id", c.RoomID)
	}
}

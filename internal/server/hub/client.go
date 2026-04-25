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

// Run starts the client's read and write pumps and returns a channel of raw
// incoming chat messages. The channel is closed when the connection ends.
// Typing events and protocol errors are handled internally and never emitted.
func (c *Client) Run(session *Session) <-chan []byte {
	incoming := make(chan []byte)
	ctx, cancel := context.WithCancel(context.Background())
	go c.writePump(ctx)
	go func() {
		defer cancel()
		defer close(incoming)
		c.readPump(ctx, session, incoming)
	}()
	return incoming
}

func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
		slog.Debug("message sent to client", "user_id", c.UserID, "room_id", c.RoomID)
	default:
		slog.Warn("client send buffer full, dropping message", "user_id", c.UserID, "room_id", c.RoomID)
	}
}

func (c *Client) SendRaw(msg []byte) {
	select {
	case c.send <- msg:
	default:
		slog.Warn("client send buffer full, dropping message", "user_id", c.UserID, "room_id", c.RoomID)
	}
}

func (c *Client) SendError(msg string) {
	errMsg := &Message{
		Type:      MessageTypeError,
		Content:   msg,
		Timestamp: time.Now(),
	}
	if b, err := errMsg.Marshal(); err == nil {
		c.Send(b)
	}
}

func (c *Client) readPump(ctx context.Context, session *Session, incoming chan<- []byte) {
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

		select {
		case incoming <- data:
		case <-ctx.Done():
			return
		}
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

package hub

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/coder/websocket"
	"github.com/egreerdp/chatatui/internal/repository"
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

func (c *Client) Run(room *Room, msgRepo *repository.MessageRepository) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.writePump(ctx)
	c.readPump(ctx, room, msgRepo) // blocking
}

func (c *Client) readPump(ctx context.Context, room *Room, msgRepo *repository.MessageRepository) {
	defer func() { _ = c.conn.CloseNow() }()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}

		var peek WireMessage
		if json.Unmarshal(data, &peek) == nil && peek.Type == MessageTypeTyping {
			typingWire := &WireMessage{
				Type:      MessageTypeTyping,
				Author:    c.Username,
				Timestamp: time.Now(),
			}
			typingBytes, err := typingWire.Marshal()
			if err != nil {
				log.Println("failed to marshal typing event:", err)
				continue
			}
			room.Broadcast(typingBytes, c)
			continue
		}

		msg := &repository.Message{
			Content:  data,
			SenderID: c.UserID,
			RoomID:   c.RoomID,
		}
		if err := msgRepo.Create(msg); err != nil {
			log.Println("failed to persist message:", err)
		}

		wire := &WireMessage{
			Type:    MessageTypeChat,
			ID:      msg.ID.String(),
			Author:  c.Username,
			Content: string(data),
		}
		if msg.CreatedAt.IsZero() {
			wire.Timestamp = time.Now()
		} else {
			wire.Timestamp = msg.CreatedAt
		}

		wireBytes, err := wire.Marshal()
		if err != nil {
			log.Println("failed to marshal message:", err)
			continue
		}

		room.Broadcast(wireBytes, c)
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
		log.Println(string(msg))
	default:
	}
}

func (c *Client) SendRaw(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}

package hub

import (
	"context"
	"log"

	"github.com/coder/websocket"
	"github.com/egreerdp/chatatui/internal/repository"
)

type Client struct {
	conn     *websocket.Conn
	roomID   string
	send     chan []byte
	UserID   uint
	DBRoomID uint
	Username string
}

func NewClient(conn *websocket.Conn, roomID string, userID, dbRoomID uint, username string) *Client {
	return &Client{
		conn:     conn,
		roomID:   roomID,
		send:     make(chan []byte, 256),
		UserID:   userID,
		DBRoomID: dbRoomID,
		Username: username,
	}
}

func (c *Client) Run(room *Room, msgRepo *repository.MessageRepository) {
	ctx, cancel := context.WithCancel(context.Background())

	go c.writePump(ctx)
	c.readPump(ctx, room, msgRepo) // blocking
	cancel()
}

func (c *Client) readPump(ctx context.Context, room *Room, msgRepo *repository.MessageRepository) {
	defer func() { _ = c.conn.CloseNow() }()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}

		// Persist the message
		msg := &repository.Message{
			Content:  data,
			SenderID: c.UserID,
			RoomID:   c.DBRoomID,
		}
		if err := msgRepo.Create(msg); err != nil {
			log.Println("failed to persist message:", err)
		}

		room.Broadcast(data, c)
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
	case c.send <- c.FormatMessageText(msg):
		log.Println(string(c.FormatMessageText(msg)))
	default:
	}
}

func (c *Client) SendRaw(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}

func (c *Client) FormatMessageText(msg []byte) []byte {
	return FormatMessageWithAuthor(msg, c.Username)
}

func FormatMessageWithAuthor(msg []byte, author string) []byte {
	return []byte(author + ": " + string(msg))
}

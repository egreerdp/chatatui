package hub

import (
	"context"

	"github.com/coder/websocket"
)

type Client struct {
	conn   *websocket.Conn
	roomID string
	send   chan []byte
}

func NewClient(conn *websocket.Conn, roomID string) *Client {
	return &Client{
		conn:   conn,
		roomID: roomID,
		send:   make(chan []byte, 256),
	}
}

func (c *Client) Run(room *Room) {
	ctx, cancel := context.WithCancel(context.Background())

	go c.writePump(ctx)
	c.readPump(ctx, room)
	cancel()
}

func (c *Client) readPump(ctx context.Context, room *Room) {
	defer func() { _ = c.conn.CloseNow() }()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
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
	case c.send <- msg:
	default:
	}
}

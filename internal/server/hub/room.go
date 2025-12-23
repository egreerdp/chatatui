package hub

import (
	"context"
	"sync"

	"github.com/coder/websocket"
)

type Room struct {
	ID      uint
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

func NewRoom() *Room {
	return &Room{
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *Room) Add(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
}

func (h *Room) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

func (h *Room) Broadcast(ctx context.Context, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		_ = conn.Write(ctx, websocket.MessageText, msg)
	}
}

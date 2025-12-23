package hub

import (
	"context"
	"sync"

	"github.com/coder/websocket"
)

type Hub struct {
	rooms map[uint]*Room
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[uint]*Room),
	}
}

func (h *Hub) Add(room *Room) {
	h.mu.Lock()
	h.rooms[uint(room.ID)] = room
	h.mu.Unlock()
}

func (h *Hub) Remove(roomID int) {
	h.mu.Lock()
	delete(h.rooms, uint(roomID))
	h.mu.Unlock()
}

func (h *Hub) Broadcast(ctx context.Context, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, room := range h.rooms {
		for conn := range room.clients {
			_ = conn.Write(ctx, websocket.MessageText, msg)
		}
	}
}

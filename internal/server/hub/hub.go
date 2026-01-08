package hub

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	Rooms map[uuid.UUID]*Room
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Rooms: make(map[uuid.UUID]*Room),
	}
}

func (h *Hub) GetOrCreateRoom(roomUUID uuid.UUID) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.Rooms[roomUUID]; ok {
		return room, nil
	}

	room := NewRoom()
	room.ID = roomUUID

	h.Rooms[roomUUID] = room

	return room, nil
}

func (h *Hub) Add(room *Room) {
	h.mu.Lock()
	h.Rooms[room.ID] = room
	h.mu.Unlock()
}

func (h *Hub) Remove(roomID uuid.UUID) {
	h.mu.Lock()
	room := h.Rooms[roomID]
	delete(h.Rooms, roomID)
	h.mu.Unlock()

	// Clean up worker pool if room exists
	if room != nil {
		room.Shutdown()
	}
}

func (h *Hub) Broadcast(msg []byte, sender *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, room := range h.Rooms {
		room.Broadcast(msg, sender)
	}
}

// Shutdown gracefully shuts down all rooms and their worker pools
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, room := range h.Rooms {
		room.Shutdown()
	}
}

package hub

import (
	"fmt"
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

func (h *Hub) GetOrCreateRoom(roomID string) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, fmt.Errorf("parse uuid: %w", err)
	}

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
	delete(h.Rooms, roomID)
	h.mu.Unlock()
}

func (h *Hub) Broadcast(msg []byte, sender *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, room := range h.Rooms {
		room.Broadcast(msg, sender)
	}
}

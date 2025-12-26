package hub

import (
	"context"
	"fmt"
	"sync"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const defaultRoomID = "aa8c0638-0cfc-4f84-8519-f2bc88474efc"

type Hub struct {
	Rooms map[uuid.UUID]*Room
	mu    sync.RWMutex
}

func NewHub() *Hub {
	defaultRoom := NewRoom()
	defaultRoom.ID = uuid.MustParse(defaultRoomID)

	rooms := make(map[uuid.UUID]*Room)
	rooms[defaultRoom.ID] = defaultRoom

	return &Hub{
		Rooms: rooms,
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

func (h *Hub) Broadcast(ctx context.Context, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, room := range h.Rooms {
		for conn := range room.clients {
			_ = conn.Write(ctx, websocket.MessageText, msg)
		}
	}
}

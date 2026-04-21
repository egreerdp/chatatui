package hub

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomExists   = errors.New("room already active")
)

type Hub struct {
	active map[uuid.UUID]*Room
	mu     sync.RWMutex
	broker Broker
	subs   map[uuid.UUID]func()
}

func NewHub(broker Broker) *Hub {
	return &Hub{
		active: make(map[uuid.UUID]*Room),
		broker: broker,
		subs:   make(map[uuid.UUID]func()),
	}
}

func (h *Hub) ActiveCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.active)
}

func (h *Hub) CreateRoom(roomUUID uuid.UUID) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.active[roomUUID]; ok {
		return nil, ErrRoomExists
	}

	publish := func(ctx context.Context, msg []byte) error {
		return h.broker.Publish(ctx, roomUUID, msg)
	}

	room := NewRoom(publish)
	room.ID = roomUUID
	h.active[roomUUID] = room

	ch, unsub, err := h.broker.Subscribe(context.Background(), roomUUID)
	if err != nil {
		slog.Error("broker subscribe failed", "room_id", roomUUID, "error", err)
		h.subs[roomUUID] = func() {}
	} else {
		h.subs[roomUUID] = unsub
		go func() {
			for msg := range ch {
				room.fanOut(msg)
			}
		}()
	}

	return room, nil
}

func (h *Hub) GetRoom(roomUUID uuid.UUID) (*Room, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.active[roomUUID]
	if !ok {
		return nil, ErrRoomNotFound
	}

	return room, nil
}

func (h *Hub) Add(room *Room) {
	h.mu.Lock()
	h.active[room.ID] = room
	h.mu.Unlock()
}

func (h *Hub) Remove(roomID uuid.UUID) {
	h.mu.Lock()
	room := h.active[roomID]
	if unsub, ok := h.subs[roomID]; ok {
		unsub()
		delete(h.subs, roomID)
	}
	delete(h.active, roomID)
	h.mu.Unlock()

	if room != nil {
		room.Shutdown()
	}
}

func (h *Hub) Publish(ctx context.Context, roomID uuid.UUID, msg []byte) error {
	return h.broker.Publish(ctx, roomID, msg)
}

func (h *Hub) Broadcast(msg []byte, sender *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, room := range h.active {
		room.Broadcast(msg, sender)
	}
}

func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, unsub := range h.subs {
		unsub()
		delete(h.subs, id)
	}

	for _, room := range h.active {
		room.Shutdown()
	}
}

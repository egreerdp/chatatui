package hub

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExists   = errors.New("session already active")
)

type Hub struct {
	active map[uuid.UUID]*Session
	mu     sync.RWMutex
	broker Broker
	subs   map[uuid.UUID]func()
}

func NewHub(broker Broker) *Hub {
	return &Hub{
		active: make(map[uuid.UUID]*Session),
		broker: broker,
		subs:   make(map[uuid.UUID]func()),
	}
}

func (h *Hub) ActiveCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.active)
}

func (h *Hub) CreateSession(roomUUID uuid.UUID) (*Session, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.active[roomUUID]; ok {
		return nil, ErrSessionExists
	}

	publish := func(ctx context.Context, msg []byte) error {
		return h.broker.Publish(ctx, roomUUID, msg)
	}

	session := NewSession(publish)
	session.ID = roomUUID
	h.active[roomUUID] = session

	ch, unsub, err := h.broker.Subscribe(context.Background(), roomUUID)
	if err != nil {
		slog.Error("broker subscribe failed", "room_id", roomUUID, "error", err)
		h.subs[roomUUID] = func() {}
	} else {
		h.subs[roomUUID] = unsub
		go func() {
			for msg := range ch {
				session.fanOut(msg)
			}
		}()
	}

	return session, nil
}

func (h *Hub) GetOrCreateSession(roomUUID uuid.UUID) (*Session, error) {
	if session, err := h.GetSession(roomUUID); err == nil {
		return session, nil
	}
	if session, err := h.CreateSession(roomUUID); !errors.Is(err, ErrSessionExists) {
		return session, err
	}
	return h.GetSession(roomUUID)
}

func (h *Hub) GetSession(roomUUID uuid.UUID) (*Session, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	session, ok := h.active[roomUUID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

func (h *Hub) Add(session *Session) {
	h.mu.Lock()
	h.active[session.ID] = session
	h.mu.Unlock()
}

func (h *Hub) Remove(roomID uuid.UUID) {
	h.mu.Lock()
	session := h.active[roomID]
	if unsub, ok := h.subs[roomID]; ok {
		unsub()
		delete(h.subs, roomID)
	}
	delete(h.active, roomID)
	h.mu.Unlock()

	if session != nil {
		session.Shutdown()
	}
}

func (h *Hub) Publish(ctx context.Context, roomID uuid.UUID, msg []byte) error {
	return h.broker.Publish(ctx, roomID, msg)
}

func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, unsub := range h.subs {
		unsub()
		delete(h.subs, id)
	}

	for _, session := range h.active {
		session.Shutdown()
	}
}

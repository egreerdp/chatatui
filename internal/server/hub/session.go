package hub

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type Session struct {
	ID            uuid.UUID
	clients       map[*Client]bool
	mu            sync.RWMutex
	broadcastPool *BroadcastPool
	poolThreshold int
	workerCount   int
	publish       func(ctx context.Context, msg []byte) error
}

func NewSession(publish func(ctx context.Context, msg []byte) error) *Session {
	return &Session{
		ID:            uuid.New(),
		clients:       make(map[*Client]bool),
		poolThreshold: 10,
		workerCount:   10,
		publish:       publish,
	}
}

func (s *Session) AddClient(c *Client) {
	s.mu.Lock()
	s.clients[c] = true
	clientCount := len(s.clients)
	s.mu.Unlock()

	if clientCount >= s.poolThreshold && s.broadcastPool == nil {
		s.activatePool()
	}
}

func (s *Session) RemoveClient(c *Client) {
	s.mu.Lock()
	delete(s.clients, c)
	close(c.send)
	s.mu.Unlock()
}

func (s *Session) Broadcast(msg []byte, _ *Client) {
	if err := s.publish(context.Background(), msg); err != nil {
		slog.Error("broker publish failed", "room_id", s.ID, "error", err)
	}
}

func (s *Session) Shutdown() {
	if s.broadcastPool != nil {
		s.broadcastPool.Shutdown()
	}
}

func (s *Session) activatePool() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.broadcastPool != nil {
		return
	}

	s.broadcastPool = NewBroadcastPool(s.workerCount)
	s.broadcastPool.Start()
	slog.Info("activated broadcast pool", "room_id", s.ID, "workers", s.workerCount)
}

func (s *Session) fanOut(msg []byte) {
	s.mu.RLock()
	poolEnabled := s.broadcastPool != nil

	clientSnapshot := make([]*Client, 0, len(s.clients))
	for client := range s.clients {
		clientSnapshot = append(clientSnapshot, client)
	}
	s.mu.RUnlock()

	if !poolEnabled {
		for _, client := range clientSnapshot {
			client.SendRaw(msg)
		}
		return
	}

	job := &broadcastJob{
		message: msg,
		clients: clientSnapshot,
	}
	s.broadcastPool.Submit(job)
}

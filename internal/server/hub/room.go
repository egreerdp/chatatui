package hub

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type Room struct {
	ID            uuid.UUID
	clients       map[*Client]bool
	mu            sync.RWMutex
	broadcastPool *BroadcastPool
	poolThreshold int
	workerCount   int
	publish       func(ctx context.Context, msg []byte) error
}

func NewRoom(publish func(ctx context.Context, msg []byte) error) *Room {
	return &Room{
		ID:            uuid.New(),
		clients:       make(map[*Client]bool),
		poolThreshold: 10,
		workerCount:   10,
		publish:       publish,
	}
}

func (r *Room) Add(c *Client) {
	r.mu.Lock()
	r.clients[c] = true
	clientCount := len(r.clients)
	r.mu.Unlock()

	if clientCount >= r.poolThreshold && r.broadcastPool == nil {
		r.activatePool()
	}
}

func (r *Room) activatePool() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.broadcastPool != nil {
		return
	}

	r.broadcastPool = NewBroadcastPool(r.workerCount)
	r.broadcastPool.Start()
	slog.Info("activated broadcast pool", "room_id", r.ID, "workers", r.workerCount)
}

func (r *Room) Remove(c *Client) {
	r.mu.Lock()
	delete(r.clients, c)
	close(c.send)
	r.mu.Unlock()
}

func (r *Room) Broadcast(msg []byte, _ *Client) {
	if err := r.publish(context.Background(), msg); err != nil {
		slog.Error("broker publish failed", "room_id", r.ID, "error", err)
	}
}

func (r *Room) fanOut(msg []byte) {
	r.mu.RLock()
	poolEnabled := r.broadcastPool != nil

	clientSnapshot := make([]*Client, 0, len(r.clients))
	for client := range r.clients {
		clientSnapshot = append(clientSnapshot, client)
	}
	r.mu.RUnlock()

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
	r.broadcastPool.Submit(job)
}

func (r *Room) Shutdown() {
	if r.broadcastPool != nil {
		r.broadcastPool.Shutdown()
	}
}

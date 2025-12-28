package hub

import (
	"sync"

	"github.com/google/uuid"
)

type Room struct {
	ID      uuid.UUID
	clients map[*Client]bool
	mu      sync.RWMutex
}

func NewRoom() *Room {
	return &Room{
		ID:      uuid.New(),
		clients: make(map[*Client]bool),
	}
}

func (r *Room) Add(c *Client) {
	r.mu.Lock()
	r.clients[c] = true
	r.mu.Unlock()
}

func (r *Room) Remove(c *Client) {
	r.mu.Lock()
	delete(r.clients, c)
	close(c.send)
	r.mu.Unlock()
}

func (r *Room) Broadcast(msg []byte, sender *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	formatted := FormatMessageWithAuthor(msg, sender.Username)

	for client := range r.clients {
		if client == sender {
			continue
		}

		client.SendRaw(formatted)
	}
}

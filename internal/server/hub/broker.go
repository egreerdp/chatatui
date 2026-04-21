package hub

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Broker interface {
	Publish(ctx context.Context, roomID uuid.UUID, msg []byte) error
	Subscribe(ctx context.Context, roomID uuid.UUID) (<-chan []byte, func(), error)
}

// RedisBroker implements Broker using Redis pub/sub.
type RedisBroker struct {
	client *redis.Client
}

func NewRedisBroker(client *redis.Client) *RedisBroker {
	return &RedisBroker{client: client}
}

func (b *RedisBroker) Publish(ctx context.Context, roomID uuid.UUID, msg []byte) error {
	return b.client.Publish(ctx, redisChannel(roomID), msg).Err()
}

func (b *RedisBroker) Subscribe(ctx context.Context, roomID uuid.UUID) (<-chan []byte, func(), error) {
	pubsub := b.client.Subscribe(ctx, redisChannel(roomID))

	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, nil, fmt.Errorf("redis subscribe: %w", err)
	}

	ch := make(chan []byte, 64)

	go func() {
		defer close(ch)
		for msg := range pubsub.Channel() {
			select {
			case ch <- []byte(msg.Payload):
			default:
			}
		}
	}()

	unsubscribe := func() {
		_ = pubsub.Close()
	}

	return ch, unsubscribe, nil
}

func redisChannel(roomID uuid.UUID) string {
	return "room:" + roomID.String()
}


package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/EwanGreer/cache"
	"github.com/redis/go-redis/v9"
)

type rateLimitEntry struct{}

func (r rateLimitEntry) CacheKey() string    { return "" }
func (r rateLimitEntry) CachePrefix() string { return "ratelimit" }

type RateLimiter struct {
	cache      cache.RedisCache[rateLimitEntry]
	maxReqs    int64
	windowSecs int
}

func NewRateLimiter(redisURL string, maxReqs, windowSecs int) (*RateLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	c, err := cache.NewCache[rateLimitEntry](client, time.Duration(windowSecs)*time.Second, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &RateLimiter{
		cache:      c,
		maxReqs:    int64(maxReqs),
		windowSecs: windowSecs,
	}, nil
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			next.ServeHTTP(w, r)
			return
		}

		key := fmt.Sprintf("ratelimit:%d", user.ID)

		allowed, err := rl.isAllowed(r.Context(), key)
		if err != nil {
			http.Error(w, "rate limit check failed", http.StatusInternalServerError)
			return
		}

		if !allowed {
			log.Println("Ratelimitted user:", user.Name)
			w.Header().Set("Retry-After", fmt.Sprintf("%d", rl.windowSecs))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) isAllowed(ctx context.Context, key string) (bool, error) {
	count, err := rl.cache.Incr(ctx, key)
	if err != nil {
		return false, err
	}

	if count == 1 {
		_, err := rl.cache.Expire(ctx, key, time.Duration(rl.windowSecs)*time.Second)
		if err != nil {
			return false, err
		}
	}

	return count <= rl.maxReqs, nil
}

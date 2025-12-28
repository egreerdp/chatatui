package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/egreerdp/chatatui/internal/repository"
)

type contextKey string

const userContextKey contextKey = "user"

func APIKeyAuth(users *repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization required", http.StatusUnauthorized)
				return
			}

			apiKey := strings.TrimPrefix(authHeader, "Bearer ")

			user, err := users.GetByAPIKey(apiKey)
			if err != nil {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) *repository.User {
	user, _ := ctx.Value(userContextKey).(*repository.User)
	return user
}

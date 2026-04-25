package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EwanGreer/chatatui/internal/domain"
)

type contextKey string

const userContextKey contextKey = "user"

type UserLookup interface {
	GetByAPIKey(apiKey string) (*domain.User, error)
}

func APIKeyAuth(users UserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "authorization required")
				return
			}

			apiKey := strings.TrimPrefix(authHeader, "Bearer ")

			user, err := users.GetByAPIKey(apiKey)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "INVALID_API_KEY", "invalid api key")
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) *domain.User {
	user, _ := ctx.Value(userContextKey).(*domain.User)
	return user
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}{Error: message, Code: code})
}

package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/EwanGreer/chatatui/internal/repository"
)

type UserStore interface {
	Create(user *domain.User) error
}

type RegisterHandler struct {
	users UserStore
}

func NewRegisterHandler(users UserStore) *RegisterHandler {
	return &RegisterHandler{users: users}
}

type registerRequest struct {
	Name string `json:"name"`
}

type registerResponse struct {
	APIKey string `json:"api_key"`
}

func (h *RegisterHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "NAME_REQUIRED", "name is required")
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate api key")
		return
	}

	user := &domain.User{
		Name:   req.Name,
		APIKey: repository.HashAPIKey(apiKey),
	}

	if err := h.users.Create(user); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(registerResponse{APIKey: apiKey})
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

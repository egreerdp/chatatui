package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/google/uuid"
)

type RegisterHandler struct {
	db *repository.PostgresDB
}

func NewRegisterHandler(db *repository.PostgresDB) *RegisterHandler {
	return &RegisterHandler{db: db}
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		http.Error(w, "failed to generate api key", http.StatusInternalServerError)
		return
	}

	user := &repository.User{
		UUID:   uuid.New(),
		Name:   req.Name,
		APIKey: apiKey,
	}

	if err := h.db.Users().Create(user); err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
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

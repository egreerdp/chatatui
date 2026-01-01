package api

import (
	"encoding/json"
	"net/http"

	"github.com/egreerdp/chatatui/internal/repository"
)

type RoomsHandler struct {
	db        *repository.PostgresDB
	listLimit int
}

func NewRoomsHandler(db *repository.PostgresDB, listLimit int) *RoomsHandler {
	return &RoomsHandler{db: db, listLimit: listLimit}
}

type roomResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *RoomsHandler) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.db.Rooms().List(h.listLimit, 0)
	if err != nil {
		http.Error(w, "failed to list rooms", http.StatusInternalServerError)
		return
	}

	resp := make([]roomResponse, len(rooms))
	for i, room := range rooms {
		name := room.Name
		if name == "" {
			name = room.UUID.String()[:8]
		}
		resp[i] = roomResponse{
			ID:   room.UUID.String(),
			Name: name,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

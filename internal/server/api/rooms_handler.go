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

type createRoomRequest struct {
	Name string `json:"name"`
}

func (h *RoomsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "NAME_REQUIRED", "room name is required")
		return
	}

	room := &repository.Room{
		Name: req.Name,
	}

	if err := h.db.Rooms().Create(room); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create room")
		return
	}

	resp := roomResponse{
		ID:   room.ID.String(),
		Name: room.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *RoomsHandler) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.db.Rooms().List(h.listLimit, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list rooms")
		return
	}

	resp := make([]roomResponse, len(rooms))
	for i, room := range rooms {
		name := room.Name
		if name == "" {
			name = room.ID.String()[:8]
		}
		resp[i] = roomResponse{
			ID:   room.ID.String(),
			Name: name,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/EwanGreer/chatatui/internal/limits"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RoomStore interface {
	Create(room *domain.Room) error
	List(limit, offset int) ([]domain.Room, error)
	GetByID(id uuid.UUID) (*domain.Room, error)
	ListRoomMembers(roomID uuid.UUID) ([]domain.RoomMember, error)
}

type RoomsHandler struct {
	rooms     RoomStore
	listLimit int
}

func NewRoomsHandler(rooms RoomStore, listLimit int) *RoomsHandler {
	return &RoomsHandler{rooms: rooms, listLimit: listLimit}
}

type RoomMemberResponse struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	LastConnectedAt *time.Time `json:"last_connected_at"`
}

type GetRoomResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
	MemberCount int                  `json:"member_count"`
	Members     []RoomMemberResponse `json:"members"`
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

func (h *RoomsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "NAME_REQUIRED", "room name is required")
		return
	}

	if len(req.Name) > limits.MaxRoomNameLength {
		writeError(w, http.StatusBadRequest, "NAME_TOO_LONG", fmt.Sprintf("room name must be %d characters or fewer", limits.MaxRoomNameLength))
		return
	}

	room := &domain.Room{Name: req.Name}

	if err := h.rooms.Create(room); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create room")
		return
	}

	resp := GetRoomResponse{
		ID:          room.ID.String(),
		Name:        room.Name,
		CreatedAt:   room.CreatedAt.String(),
		UpdatedAt:   room.UpdatedAt.String(),
		MemberCount: 0,
		Members:     []RoomMemberResponse{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *RoomsHandler) Index(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.rooms.List(h.listLimit, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list rooms")
		return
	}

	resp := make([]GetRoomResponse, len(rooms))
	for i, room := range rooms {
		name := room.Name
		if name == "" {
			name = room.ID.String()[:8]
		}
		members := make([]RoomMemberResponse, len(room.Members))
		for j, m := range room.Members {
			members[j] = RoomMemberResponse{
				ID:   m.UserID.String(),
				Name: m.Name,
			}
		}
		resp[i] = GetRoomResponse{
			ID:          room.ID.String(),
			Name:        name,
			CreatedAt:   room.CreatedAt.String(),
			UpdatedAt:   room.UpdatedAt.String(),
			MemberCount: len(members),
			Members:     members,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *RoomsHandler) Show(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ROOM_ID", "invalid room id")
		return
	}

	room, err := h.rooms.GetByID(roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "room not found")
		return
	}

	roomMembers, err := h.rooms.ListRoomMembers(roomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get room members")
		return
	}

	members := make([]RoomMemberResponse, len(roomMembers))
	for i, m := range roomMembers {
		var lastConnectedAt *time.Time
		if !m.LastConnectedAt.IsZero() {
			lastConnectedAt = &m.LastConnectedAt
		}

		members[i] = RoomMemberResponse{
			ID:              m.UserID.String(),
			Name:            m.Name,
			LastConnectedAt: lastConnectedAt,
		}
	}

	name := room.Name
	if name == "" {
		name = room.ID.String()[:8]
	}

	resp := GetRoomResponse{
		ID:          room.ID.String(),
		Name:        name,
		CreatedAt:   room.CreatedAt.String(),
		UpdatedAt:   room.UpdatedAt.String(),
		MemberCount: len(members),
		Members:     members,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

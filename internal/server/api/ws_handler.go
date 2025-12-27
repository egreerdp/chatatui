package api

import (
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type WSHandler struct {
	hub *hub.Hub
	db  *repository.SQLiteDB
}

func NewWSHandler(h *hub.Hub, db *repository.SQLiteDB) *WSHandler {
	go func() {
		for {
			time.Sleep(time.Second * 5)

			log.Println("Room Count:", len(h.Rooms))
		}
	}()

	return &WSHandler{
		hub: h,
		db:  db,
	}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		http.Error(w, "room required", http.StatusBadRequest)
		return
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	dbRoom, err := h.db.Rooms().GetOrCreateByUUID(roomUUID)
	if err != nil {
		log.Println("GetOrCreateRoomByUUID: err:", err)
		http.Error(w, "failed to get room", http.StatusInternalServerError)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		log.Println("err:", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	userUUID := uuid.New()
	userName := "anonymous"
	dbUser, err := h.db.Users().GetOrCreateByUUID(userUUID, userName)
	if err != nil {
		log.Println("failed to create user:", err)
		_ = conn.Close(websocket.StatusInternalError, "failed to create user")
		return
	}

	// Add user as room member
	if err := h.db.Rooms().AddMember(dbRoom.ID, dbUser.ID); err != nil {
		log.Println("failed to add room member:", err)
	}

	client := hub.NewClient(conn, roomID, dbUser.ID, dbRoom.ID)

	room, err := h.hub.GetOrCreateRoom(roomID)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "failed to join room")
		return
	}

	room.Add(client)
	defer room.Remove(client)

	// Send message history to the client
	h.sendHistory(client, dbRoom.ID)

	client.Run(room, h.db.Messages())
}

func (h *WSHandler) sendHistory(client *hub.Client, roomID uint) {
	messages, err := h.db.Messages().GetByRoom(roomID, 50, 0)
	if err != nil {
		log.Println("failed to get message history:", err)
		return
	}

	// Send messages in chronological order (oldest first)
	for i := len(messages) - 1; i >= 0; i-- {
		client.Send(messages[i].Content)
	}
}

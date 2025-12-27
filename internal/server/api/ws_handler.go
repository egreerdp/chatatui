package api

import (
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/go-chi/chi/v5"
)

type WSHandler struct {
	hub *hub.Hub
}

func NewWSHandler(h *hub.Hub) *WSHandler {
	go func() {
		for {
			time.Sleep(time.Second * 5)

			log.Println("Room Count:", len(h.Rooms))
		}
	}()

	return &WSHandler{
		hub: h,
	}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		http.Error(w, "room required", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		log.Println("err:", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	client := hub.NewClient(conn, roomID)

	room, err := h.hub.GetOrCreateRoom(roomID)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "failed to join room")
		return
	}

	room.Add(client)
	defer room.Remove(client)

	client.Run(room)
}

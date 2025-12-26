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
		log.Println("roomID:", roomID)
		log.Println("URL:", r.URL)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()

	room, err := h.hub.GetOrCreateRoom(roomID)
	if err != nil {
		log.Println("get or create room")
		return
	}

	room.Add(conn)
	defer room.Remove(conn)

	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		room.Broadcast(r.Context(), data)
	}
}

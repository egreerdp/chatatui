package api

import (
	"net/http"

	"github.com/coder/websocket"
	"github.com/egreerdp/chatatui/internal/server/hub"
)

type WSHandler struct {
	hub *hub.Hub
}

func NewWSHandler(h *hub.Hub) *WSHandler {
	return &WSHandler{
		hub: h,
	}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()

	h.hub.Add(conn)
	defer h.hub.Remove(conn)

	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		h.hub.Broadcast(r.Context(), data)
	}
}

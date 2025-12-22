package server

import (
	"context"
	"log"
	"net/http"

	"github.com/egreerdp/chatatui/internal/server/api"
)

type ChatServer struct {
	handler *api.Handler
	srv     *http.Server
	addr    string
}

func NewChatServer(h *api.Handler, addr string) *ChatServer {
	return &ChatServer{
		handler: h,
		addr:    addr,
	}
}

func (cs *ChatServer) Start() error {
	cs.srv = &http.Server{
		Addr:    cs.addr,
		Handler: cs.handler.Router,
	}

	log.Println("listen and serve")
	return cs.srv.ListenAndServe()
}

func (cs *ChatServer) Stop() error {
	if cs.srv != nil {
		return cs.srv.Shutdown(context.Background())
	}

	return nil
}

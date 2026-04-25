package server

import (
	"context"
	"net/http"

	"github.com/EwanGreer/chatatui/internal/server/api"
)

type ChatServer struct {
	handler    *api.Handler
	srv        *http.Server
	addr       string
	onShutdown func()
}

func NewChatServer(h *api.Handler, addr string, onShutdown func()) *ChatServer {
	return &ChatServer{
		handler:    h,
		addr:       addr,
		onShutdown: onShutdown,
	}
}

func (cs *ChatServer) Start() error {
	cs.srv = &http.Server{
		Addr:    cs.addr,
		Handler: cs.handler.Routes(),
	}

	return cs.srv.ListenAndServe()
}

func (cs *ChatServer) Stop(ctx context.Context) error {
	if cs.onShutdown != nil {
		cs.onShutdown()
	}

	if cs.srv != nil {
		return cs.srv.Shutdown(ctx)
	}

	return nil
}

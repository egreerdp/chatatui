package server

import "github.com/egreerdp/chatatui/internal/server/api"

type ChatServer struct {
	handler *api.Handler
}

func NewChatServer(h *api.Handler) *ChatServer {
	return &ChatServer{
		handler: h,
	}
}

func (cs *ChatServer) Start() error {
	return nil
}

func (cs *ChatServer) Stop() error {
	return nil
}

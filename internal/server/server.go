package server

type ChatServer struct{}

func NewChatServer() *ChatServer {
	return &ChatServer{}
}

func (cs *ChatServer) Start() error {
	return nil
}

func (cs *ChatServer) Stop() error {
	return nil
}

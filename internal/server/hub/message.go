package hub

import "github.com/EwanGreer/chatatui/internal/domain"

// Type aliases so all hub-internal code keeps compiling without changes.
type MessageType = domain.WireMessageType
type Message = domain.WireMessage

const (
	MessageTypeChat   = domain.WireMessageTypeChat
	MessageTypeSystem = domain.WireMessageTypeSystem
	MessageTypeTyping = domain.WireMessageTypeTyping
	MessageTypeError  = domain.WireMessageTypeError
)

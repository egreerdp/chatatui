package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/EwanGreer/chatatui/internal/limits"
	"github.com/EwanGreer/chatatui/internal/server/hub"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/coder/websocket"
)

type focus int

const (
	focusRooms focus = iota
	focusMessages
	focusInput
	focusCreateRoom
)

type connState int

const (
	connStateDisconnected connState = iota
	connStateConnecting
	connStateConnected
)

type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Config struct {
	ServerAddr string
	APIKey     string
}

type Model struct {
	config          Config
	viewport        viewport.Model
	input           textinput.Model
	createRoomInput textinput.Model
	rooms           []Room
	messages        []string
	focus           focus
	width           int
	height          int
	ready           bool
	roomIndex       int
	err             error
	conn            *websocket.Conn
	connectedTo     string
	state           connState
	reconnectDelay  time.Duration
	typingUsers     map[string]time.Time
	lastTypingSent  time.Time
}

type (
	roomsMsg     []Room
	errMsg       error
	connectedMsg struct {
		roomID string
		conn   *websocket.Conn
	}
	roomCreatedMsg Room
	tickMsg        time.Time
	reconnectMsg   string
)

type incomingMsg struct {
	formatted string
	author    string
}

type typingMsg string // username of the person who is typing

type wireMessage struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

func NewModel(cfg Config) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = limits.MaxMessageLength
	ti.Focus()

	createInput := textinput.New()
	createInput.Placeholder = "Enter room name..."
	createInput.CharLimit = limits.MaxRoomNameLength
	createInput.Width = 30

	return &Model{
		config:          cfg,
		input:           ti,
		createRoomInput: createInput,
		rooms:           []Room{},
		messages:        []string{},
		focus:           focusInput,
		reconnectDelay:  time.Second,
		typingUsers:     make(map[string]time.Time),
	}
}

func (c Config) httpURL(path string) string {
	base := c.ServerAddr
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	return base + path
}

func (c Config) wsURL(path string) string {
	base := c.ServerAddr
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	default:
		base = "ws://" + base
	}
	return base + path
}

func formatWireMessage(data []byte) string {
	var wire wireMessage
	if err := json.Unmarshal(data, &wire); err != nil {
		return string(data)
	}

	ts := wire.Timestamp.Local().Format("15:04")

	if wire.Type == hub.MessageTypeError.String() {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Italic(true).
			Render(fmt.Sprintf("%s ! %s", ts, wire.Content))
	}

	return fmt.Sprintf("%s %s: %s", ts, wire.Author, wire.Content)
}

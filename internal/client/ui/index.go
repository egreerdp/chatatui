package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
)
type incomingMsg string

func NewModel(cfg Config) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	createInput := textinput.New()
	createInput.Placeholder = "Enter room name..."
	createInput.Width = 30

	return &Model{
		config:          cfg,
		input:           ti,
		createRoomInput: createInput,
		rooms:           []Room{},
		messages:        []string{},
		focus:           focusInput,
	}
}

func (m Model) fetchRooms() tea.Msg {
	url := fmt.Sprintf("http://%s/rooms", m.config.ServerAddr)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errMsg(err)
	}
	req.Header.Set("Authorization", m.config.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errMsg(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return errMsg(fmt.Errorf("server returned %d", resp.StatusCode))
	}

	var rooms []Room
	if err := json.NewDecoder(resp.Body).Decode(&rooms); err != nil {
		return errMsg(err)
	}

	return roomsMsg(rooms)
}

func (m Model) createRoom(name string) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("http://%s/rooms", m.config.ServerAddr)

		payload := map[string]string{"name": name}
		body, err := json.Marshal(payload)
		if err != nil {
			return errMsg(err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			return errMsg(err)
		}
		req.Header.Set("Authorization", m.config.APIKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errMsg(err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusCreated {
			return errMsg(fmt.Errorf("server returned %d", resp.StatusCode))
		}

		var room Room
		if err := json.NewDecoder(resp.Body).Decode(&room); err != nil {
			return errMsg(err)
		}

		return roomCreatedMsg(room)
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.fetchRooms, m.tickCmd())
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) connectToRoom(roomID string) tea.Cmd {
	return func() tea.Msg {
		if m.conn != nil {
			_ = m.conn.Close(websocket.StatusNormalClosure, "switching rooms")
		}

		url := fmt.Sprintf("ws://%s/ws/%s", m.config.ServerAddr, roomID)

		ctx := context.Background()
		conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
			HTTPHeader: http.Header{
				"Authorization": []string{m.config.APIKey},
			},
		})
		if err != nil {
			return errMsg(err)
		}

		return connectedMsg{roomID: roomID, conn: conn}
	}
}

func (m *Model) listenForMessages() tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return nil
		}

		_, data, err := m.conn.Read(context.Background())
		if err != nil {
			// Ignore normal close errors (happens when switching rooms)
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}
			return errMsg(err)
		}

		return incomingMsg(string(data))
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(m.fetchRooms, m.tickCmd())

	case roomsMsg:
		// Preserve current room index if possible
		oldSelectedID := ""
		if m.roomIndex < len(m.rooms) {
			oldSelectedID = m.rooms[m.roomIndex].ID
		}

		m.rooms = msg

		// Try to keep the same room selected
		if oldSelectedID != "" {
			for i, room := range m.rooms {
				if room.ID == oldSelectedID {
					m.roomIndex = i
					break
				}
			}
		}

		// Only auto-connect on first load (when not connected)
		if m.connectedTo == "" && len(m.rooms) > 0 {
			return m, m.connectToRoom(m.rooms[0].ID)
		}
		return m, nil

	case roomCreatedMsg:
		m.rooms = append([]Room{Room(msg)}, m.rooms...)
		m.roomIndex = 0
		m.setFocus(focusRooms)
		m.createRoomInput.Reset()
		return m, m.connectToRoom(msg.ID)

	case connectedMsg:
		m.conn = msg.conn
		m.connectedTo = msg.roomID
		m.err = nil
		m.messages = []string{}
		m.updateViewportContent()
		return m, m.listenForMessages()

	case incomingMsg:
		m.messages = append(m.messages, string(msg))
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, m.listenForMessages()

	case errMsg:
		m.err = msg
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "r":
			if m.focus == focusRooms {
				return m, m.fetchRooms
			}
		case "n":
			if m.focus == focusRooms {
				m.setFocus(focusCreateRoom)
				return m, nil
			}
		case "esc":
			if m.focus == focusCreateRoom {
				m.setFocus(focusRooms)
				m.createRoomInput.Reset()
				return m, nil
			}
			if m.focus == focusInput || m.focus == focusMessages {
				m.setFocus(focusRooms)
				return m, nil
			}
		case "left", "[":
			if m.focus == focusCreateRoom {
				return m, nil
			}
			if m.focus == focusInput || m.focus == focusMessages {
				m.setFocus(focusRooms)
			}
			return m, nil
		case "right", "]":
			if m.focus == focusCreateRoom {
				return m, nil
			}
			if m.focus == focusRooms {
				m.setFocus(focusInput)
			}
			return m, nil
		case "enter":
			if m.focus == focusCreateRoom && m.createRoomInput.Value() != "" {
				roomName := m.createRoomInput.Value()
				return m, m.createRoom(roomName)
			}
			if m.focus == focusRooms && len(m.rooms) > 0 {
				roomID := m.rooms[m.roomIndex].ID
				m.setFocus(focusInput)
				return m, m.connectToRoom(roomID)
			}
			if m.focus == focusInput && m.input.Value() != "" && m.conn != nil {
				msg := m.input.Value()
				m.input.Reset()
				err := m.conn.Write(context.Background(), websocket.MessageText, []byte(msg))
				if err != nil {
					m.err = err
				} else {
					m.messages = append(m.messages, "You: "+msg)
					m.updateViewportContent()
					m.viewport.GotoBottom()
				}
			}
		case "up", "k":
			if m.focus == focusRooms {
				if m.roomIndex > 0 {
					m.roomIndex--
				}
				return m, nil
			}
		case "down", "j":
			if m.focus == focusRooms {
				if m.roomIndex < len(m.rooms)-1 {
					m.roomIndex++
				}
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Account for outer border and help bar
		innerWidth := m.width - 4
		innerHeight := m.height - 4 - 1

		sidebarWidth := m.sidebarWidth()
		mainWidth := innerWidth - sidebarWidth - 1

		headerHeight := 1
		inputHeight := 3
		viewportHeight := innerHeight - headerHeight - inputHeight

		if !m.ready {
			m.viewport = viewport.New(mainWidth, viewportHeight)
			m.viewport.SetContent("")
			m.ready = true
		} else {
			m.viewport.Width = mainWidth
			m.viewport.Height = viewportHeight
		}

		m.input.Width = mainWidth - 2
	}

	if m.focus == focusInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.focus == focusCreateRoom {
		var cmd tea.Cmd
		m.createRoomInput, cmd = m.createRoomInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.focus == focusMessages {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	sidebar := m.renderSidebar()
	main := m.renderMain()

	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	help := m.renderHelp()

	appStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width - 2).
		Height(m.height - 2)

	view := appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content, help))

	// Show create room modal if in create mode
	if m.focus == focusCreateRoom {
		view = m.renderCreateRoomModal()
	}

	return view
}

func (m *Model) setFocus(f focus) {
	if m.focus == focusInput {
		m.input.Blur()
	}
	if m.focus == focusCreateRoom {
		m.createRoomInput.Blur()
	}
	m.focus = f
	if f == focusInput {
		m.input.Focus()
	}
	if f == focusCreateRoom {
		m.createRoomInput.Focus()
	}
}

func (m Model) sidebarWidth() int {
	return 20
}

func (m Model) renderSidebar() string {
	width := m.sidebarWidth()
	innerHeight := m.height - 4 - 1

	style := lipgloss.NewStyle().
		Width(width).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderRight(true).
		Padding(0, 1)

	if m.focus == focusRooms {
		style = style.BorderForeground(lipgloss.Color("62"))
	}

	header := lipgloss.NewStyle().Bold(true).Render("Rooms")

	var roomList string
	for i, room := range m.rooms {
		name := room.Name
		if i == m.roomIndex {
			name = "> " + name
		} else {
			name = "  " + name
		}
		roomList += name + "\n"
	}

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		roomList = errStyle.Render("Error: " + m.err.Error())
	} else if len(m.rooms) == 0 {
		roomList = "(no rooms)"
	}

	content := header + "\n\n" + roomList

	return style.Render(content)
}

func (m Model) renderMain() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	title := "chatatui"
	if m.connectedTo != "" {
		for _, room := range m.rooms {
			if room.ID == m.connectedTo {
				title = room.Name
				break
			}
		}
	}
	header := headerStyle.Render(title)

	viewportStyle := lipgloss.NewStyle()
	if m.focus == focusMessages {
		viewportStyle = viewportStyle.BorderForeground(lipgloss.Color("62"))
	}

	inputStyle := lipgloss.NewStyle().
		Padding(0, 1)

	if m.focus == focusInput {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color("62"))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		viewportStyle.Render(m.viewport.View()),
		inputStyle.Render(m.input.View()),
	)
}

func (m *Model) updateViewportContent() {
	var content strings.Builder
	for _, msg := range m.messages {
		content.WriteString(msg + "\n")
	}
	m.viewport.SetContent(content.String())
}

func (m Model) renderCreateRoomModal() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(40).
		Background(lipgloss.Color("235"))

	title := lipgloss.NewStyle().Bold(true).Render("Create New Room")
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("Press Enter to create, Esc to cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.createRoomInput.View(),
		"",
		help,
	)

	modal := modalStyle.Render(content)

	// Center the modal
	overlay := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)

	return overlay
}

func (m Model) renderHelp() string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	keys := []struct{ key, desc string }{
		{"←/→/esc", "switch panel"},
		{"j/k", "navigate"},
		{"n", "new room"},
		{"r", "refresh"},
		{"enter", "join/send"},
		{"ctrl+c", "quit"},
	}

	var items []string
	for _, k := range keys {
		items = append(items, keyStyle.Render(k.key)+" "+descStyle.Render(k.desc))
	}

	sep := descStyle.Render(" • ")

	var result strings.Builder
	for i, item := range items {
		if i > 0 {
			result.WriteString(sep)
		}
		result.WriteString(item)
	}

	return "  " + result.String()
}

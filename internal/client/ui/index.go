package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	config      Config
	viewport    viewport.Model
	input       textinput.Model
	rooms       []Room
	messages    []string
	focus       focus
	width       int
	height      int
	ready       bool
	roomIndex   int
	err         error
	conn        *websocket.Conn
	connectedTo string
}

type roomsMsg []Room
type errMsg error
type connectedMsg struct {
	roomID string
	conn   *websocket.Conn
}
type incomingMsg string

func NewModel(cfg Config) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	return &Model{
		config:   cfg,
		input:    ti,
		rooms:    []Room{},
		messages: []string{},
		focus:    focusInput,
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errMsg(fmt.Errorf("server returned %d", resp.StatusCode))
	}

	var rooms []Room
	if err := json.NewDecoder(resp.Body).Decode(&rooms); err != nil {
		return errMsg(err)
	}

	return roomsMsg(rooms)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.fetchRooms)
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
			return errMsg(err)
		}

		return incomingMsg(string(data))
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case roomsMsg:
		m.rooms = msg
		if len(m.rooms) > 0 {
			return m, m.connectToRoom(m.rooms[0].ID)
		}
		return m, nil

	case connectedMsg:
		m.conn = msg.conn
		m.connectedTo = msg.roomID
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
		case "left", "[":
			if m.focus == focusMessages {
				m.setFocus(focusRooms)
			} else if m.focus == focusInput {
				m.setFocus(focusMessages)
			}
			return m, nil
		case "right", "]":
			if m.focus == focusRooms {
				m.setFocus(focusMessages)
			} else if m.focus == focusMessages {
				m.setFocus(focusInput)
			}
			return m, nil
		case "enter":
			if m.focus == focusRooms && len(m.rooms) > 0 {
				roomID := m.rooms[m.roomIndex].ID
				if roomID != m.connectedTo {
					return m, m.connectToRoom(roomID)
				}
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

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content, help))
}

func (m *Model) setFocus(f focus) {
	if m.focus == focusInput {
		m.input.Blur()
	}
	m.focus = f
	if f == focusInput {
		m.input.Focus()
	}
}

func (m *Model) cycleFocus() {
	switch m.focus {
	case focusRooms:
		m.setFocus(focusMessages)
	case focusMessages:
		m.setFocus(focusInput)
	case focusInput:
		m.setFocus(focusRooms)
	}
}

func (m *Model) cycleFocusReverse() {
	switch m.focus {
	case focusRooms:
		m.setFocus(focusInput)
	case focusMessages:
		m.setFocus(focusRooms)
	case focusInput:
		m.setFocus(focusMessages)
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
	var content string
	for _, msg := range m.messages {
		content += msg + "\n"
	}
	m.viewport.SetContent(content)
}

func (m Model) renderHelp() string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	keys := []struct{ key, desc string }{
		{"←/→", "panels"},
		{"j/k", "navigate"},
		{"enter", "send"},
		{"ctrl+c", "quit"},
	}

	var items []string
	for _, k := range keys {
		items = append(items, keyStyle.Render(k.key)+" "+descStyle.Render(k.desc))
	}

	sep := descStyle.Render(" • ")

	var result string
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}

	return "  " + result
}

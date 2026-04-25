package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		now := time.Now()
		for user, t := range m.typingUsers {
			if now.Sub(t) >= typingUserTTL {
				delete(m.typingUsers, user)
			}
		}
		return m, tea.Batch(m.fetchRooms(), m.tickCmd())

	case roomsMsg:
		// Preserve current room index if possible
		oldSelectedID := ""
		if m.roomIndex < len(m.rooms) {
			oldSelectedID = m.rooms[m.roomIndex].ID
		}

		sort.Slice(msg, func(i, j int) bool {
			return msg[i].Name < msg[j].Name
		})
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
		m.rooms = append(m.rooms, Room(msg))
		sort.Slice(m.rooms, func(i, j int) bool {
			return m.rooms[i].Name < m.rooms[j].Name
		})
		for i, r := range m.rooms {
			if r.ID == msg.ID {
				m.roomIndex = i
				break
			}
		}
		m.setFocus(focusRooms)
		m.createRoomInput.Reset()
		return m, m.connectToRoom(msg.ID)

	case connectedMsg:
		m.conn = msg.conn
		m.connectedTo = msg.roomID
		m.state = connStateConnected
		m.reconnectDelay = time.Second
		m.messages = []string{}
		m.updateViewportContent()
		return m, m.listenForMessages()

	case incomingMsg:
		delete(m.typingUsers, msg.author)
		m.messages = append(m.messages, msg.formatted)
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, m.listenForMessages()

	case typingMsg:
		if author := string(msg); author != "" {
			m.typingUsers[author] = time.Now()
		}
		return m, m.listenForMessages()

	case reconnectMsg:
		return m, m.connectToRoom(string(msg))

	case clearFlashMsg:
		m.flash = ""
		return m, nil

	case errMsg:
		if m.connectedTo != "" {
			delay := m.reconnectDelay
			m.reconnectDelay = min(delay*2, 30*time.Second)
			m.state = connStateConnecting
			return m, tea.Tick(delay, func(t time.Time) tea.Msg {
				return reconnectMsg(m.connectedTo)
			})
		}
		m.flash = msg.Error()
		return m, clearFlashCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.focus != focusInput && m.focus != focusCreateRoom {
				return m, tea.Quit
			}
		case "tab", "shift+tab":
			if m.focus != focusCreateRoom {
				if m.focus == focusRooms {
					m.setFocus(focusInput)
				} else {
					m.setFocus(focusRooms)
				}
				return m, nil
			}
		case "r":
			if m.focus == focusRooms {
				return m, m.fetchRooms()
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
				text := m.input.Value()
				m.input.Reset()
				m.messages = append(m.messages, fmt.Sprintf("%s You: %s", time.Now().Local().Format("15:04"), text))
				m.updateViewportContent()
				m.viewport.GotoBottom()
				return m, sendMessageCmd(m.conn, text)
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
		default:
			// Send a typing event when the user is actively typing in the input
			// box. Debounced to at most once every 2 seconds.
			if m.shouldSendTyping() {
				m.lastTypingSent = time.Now()
				cmds = append(cmds, sendTypingCmd(m.conn))
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Account for outer border and help bar
		innerWidth := m.width - layoutOuterChrome
		innerHeight := m.height - layoutOuterChrome - layoutHelpBarHeight

		mainWidth := innerWidth - layoutSidebarWidth - layoutSidebarDivider
		viewportHeight := innerHeight - layoutHeaderHeight - layoutInputHeight - layoutTypingLine - layoutViewportBorder

		if !m.ready {
			m.viewport = viewport.New(mainWidth, viewportHeight)
			m.viewport.SetContent("")
			m.ready = true
		} else {
			m.viewport.Width = mainWidth
			m.viewport.Height = viewportHeight
		}

		m.input.Width = mainWidth - layoutInputPadding
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

func (m *Model) shouldSendTyping() bool {
	if m.focus != focusInput {
		return false
	}
	if m.conn == nil {
		return false
	}
	if time.Since(m.lastTypingSent) <= 2*time.Second {
		return false
	}
	return m.input.Value() != ""
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

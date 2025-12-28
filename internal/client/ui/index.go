package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusRooms focus = iota
	focusMessages
	focusInput
)

type Room struct {
	ID   string
	Name string
}

type Model struct {
	viewport  viewport.Model
	input     textinput.Model
	rooms     []Room
	messages  []string
	focus     focus
	width     int
	height    int
	ready     bool
	roomIndex int
}

func NewModel() *Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	return &Model{
		input:    ti,
		rooms:    []Room{},
		messages: []string{},
		focus:    focusInput,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
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
			if m.focus == focusInput && m.input.Value() != "" {
				m.messages = append(m.messages, "You: "+m.input.Value())
				m.input.Reset()
				m.updateViewportContent()
				m.viewport.GotoBottom()
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

	if len(m.rooms) == 0 {
		roomList = "(no rooms)"
	}

	content := header + "\n\n" + roomList

	return style.Render(content)
}

func (m Model) renderMain() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("chatatui")

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

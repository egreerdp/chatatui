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
		case "tab":
			m.cycleFocus()
		case "enter":
			if m.focus == focusInput && m.input.Value() != "" {
				m.messages = append(m.messages, "You: "+m.input.Value())
				m.input.Reset()
				m.updateViewportContent()
				m.viewport.GotoBottom()
			}
		case "up", "k":
			if m.focus == focusRooms && m.roomIndex > 0 {
				m.roomIndex--
			}
		case "down", "j":
			if m.focus == focusRooms && m.roomIndex < len(m.rooms)-1 {
				m.roomIndex++
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		sidebarWidth := m.sidebarWidth()
		mainWidth := m.width - sidebarWidth - 1

		headerHeight := 1
		inputHeight := 3
		viewportHeight := m.height - headerHeight - inputHeight

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

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m *Model) cycleFocus() {
	switch m.focus {
	case focusInput:
		m.focus = focusRooms
		m.input.Blur()
	case focusRooms:
		m.focus = focusMessages
	case focusMessages:
		m.focus = focusInput
		m.input.Focus()
	}
}

func (m Model) sidebarWidth() int {
	return 20
}

func (m Model) renderSidebar() string {
	width := m.sidebarWidth()

	style := lipgloss.NewStyle().
		Width(width).
		Height(m.height).
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

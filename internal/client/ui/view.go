package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/EwanGreer/chatatui/internal/limits"
	"github.com/charmbracelet/lipgloss"
)

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

	if m.state == connStateConnecting && m.connectedTo != "" {
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		roomList += warnStyle.Render("\nReconnecting...")
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

	var stateIndicator string
	switch m.state {
	case connStateConnected:
		stateIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Render(" ●")
	case connStateConnecting:
		stateIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(" ●")
	case connStateDisconnected:
		stateIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(" ●")
	}
	header := headerStyle.Render(title + stateIndicator)

	viewportStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
	if m.focus == focusMessages {
		viewportStyle = viewportStyle.BorderForeground(lipgloss.Color("62"))
	}

	inputStyle := lipgloss.NewStyle().
		Padding(0, 1)

	if m.focus == focusInput {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color("62"))
	}

	typingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		PaddingLeft(1)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		viewportStyle.Render(m.viewport.View()),
		m.renderStatusLine(typingStyle),
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

func (m Model) typingLine() string {
	var typers []string
	now := time.Now()
	for user, t := range m.typingUsers {
		if now.Sub(t) < 4*time.Second {
			typers = append(typers, user)
		}
	}
	sort.Strings(typers)

	switch len(typers) {
	case 0:
		return ""
	case 1:
		return typers[0] + " is typing..."
	case 2:
		return typers[0] + " and " + typers[1] + " are typing..."
	default:
		return "Several people are typing..."
	}
}

func (m Model) charCountIndicator() string {
	if m.focus != focusInput {
		return ""
	}
	count := len([]rune(m.input.Value()))
	if count == 0 {
		return ""
	}
	remaining := limits.MaxMessageLength - count
	text := fmt.Sprintf("%d/%d", count, limits.MaxMessageLength)
	switch {
	case remaining < 10:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(text)
	case remaining < 50:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(text)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(text)
	}
}

func (m Model) renderStatusLine(typingStyle lipgloss.Style) string {
	typingText := typingStyle.Render(m.typingLine())
	counter := m.charCountIndicator()
	if counter == "" {
		return typingText
	}
	gap := m.viewport.Width - lipgloss.Width(typingText) - lipgloss.Width(counter)
	if gap < 1 {
		gap = 1
	}
	return typingText + strings.Repeat(" ", gap) + counter
}

func (m Model) renderHelp() string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	keys := []struct{ key, desc string }{
		{"tab/←/→", "switch panel"},
		{"j/k", "navigate"},
		{"n", "new room"},
		{"r", "refresh"},
		{"enter", "join/send"},
		{"q/ctrl+c", "quit"},
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

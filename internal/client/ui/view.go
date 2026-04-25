package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/EwanGreer/chatatui/internal/limits"
	"github.com/EwanGreer/chatatui/internal/server/hub"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return styleMuted.Render("Loading...")
	}

	sidebar := m.renderSidebar()
	main := m.renderMain()

	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	help := m.renderHelp()

	appStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Width(m.width - 2).
		Height(m.height - 2)

	view := appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content, help))

	// Show create room modal if in create mode
	if m.focus == focusCreateRoom {
		view = m.renderCreateRoomModal()
	}

	return view
}

func (m Model) renderSidebar() string {
	width := m.sidebarWidth()
	innerHeight := m.height - layoutOuterChrome - layoutHelpBarHeight

	style := lipgloss.NewStyle().
		Width(width).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderRight(true).
		Padding(0, 1)

	if m.focus == focusRooms {
		style = style.BorderForeground(colorFocus)
	}

	header := styleBold.Render("Rooms")

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
		roomList = styleMuted.Render("(no rooms)")
	}

	if m.state == connStateConnecting && m.connectedTo != "" {
		roomList += styleWarning.Render("\nReconnecting...")
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
		stateIndicator = styleStateConnected.Render(" ●")
	case connStateConnecting:
		stateIndicator = styleStateConnecting.Render(" ●")
	case connStateDisconnected:
		stateIndicator = styleStateDisconnected.Render(" ●")
	}
	header := headerStyle.Render(title + stateIndicator)

	viewportStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted)
	if m.focus == focusMessages {
		viewportStyle = viewportStyle.BorderForeground(colorFocus)
	}

	inputStyle := lipgloss.NewStyle().
		Padding(0, 1)

	if m.focus == focusInput {
		inputStyle = inputStyle.BorderForeground(colorFocus)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		viewportStyle.Render(m.viewport.View()),
		m.renderStatusLine(),
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
		BorderForeground(colorFocus).
		Padding(1, 2).
		Width(40).
		Background(colorModalBg)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		styleModalTitle.Render("Create New Room"),
		"",
		m.createRoomInput.View(),
		"",
		styleModalHelp.Render("Press Enter to create, Esc to cancel"),
	)

	modal := modalStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

func (m Model) typingLine() string {
	var typers []string
	now := time.Now()
	for user, t := range m.typingUsers {
		if now.Sub(t) < typingUserTTL {
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
		return styleError.Render(text)
	case remaining < 50:
		return styleWarning.Render(text)
	default:
		return styleMuted.Render(text)
	}
}

func (m Model) renderStatusLine() string {
	if m.flash != "" {
		return styleError.Render("  ✕ " + m.flash)
	}
	typingText := styleTyping.Render(m.typingLine())
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
		items = append(items, styleHelpKey.Render(k.key)+" "+styleHelpDesc.Render(k.desc))
	}

	sep := styleHelpDesc.Render(" • ")

	var result strings.Builder
	for i, item := range items {
		if i > 0 {
			result.WriteString(sep)
		}
		result.WriteString(item)
	}

	return "  " + result.String()
}

func (m Model) sidebarWidth() int {
	return layoutSidebarWidth
}

func formatWireMessage(data []byte) string {
	var wire wireMessage
	if err := json.Unmarshal(data, &wire); err != nil {
		return string(data)
	}

	ts := wire.Timestamp.Local().Format("15:04")

	if wire.Type == hub.MessageTypeError.String() {
		return styleError.Italic(true).Render(fmt.Sprintf("%s ! %s", ts, wire.Content))
	}

	return fmt.Sprintf("%s %s: %s", ts, wire.Author, wire.Content)
}

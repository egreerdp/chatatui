package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct{}

func NewModel() *Model {
	return &Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString("\nPress q to quit.\n")

	return sb.String()
}

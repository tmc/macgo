package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/tmc/macgo"
)

type model struct {
	cursor int
	items  []string
	done   bool
}

func initialModel() model {
	return model{
		items: []string{"Camera", "Microphone", "Screen Recording", "Files", "Location"},
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.done {
		return fmt.Sprintf("Selected: %s\n", m.items[m.cursor])
	}

	var b strings.Builder
	b.WriteString("macgo bubbletea TUI test\n\n")
	b.WriteString("Pick a permission:\n\n")

	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, item))
	}

	b.WriteString("\nj/k or arrows to move, enter to select, q to quit\n")
	return b.String()
}

func main() {
	if err := macgo.Start(macgo.NewConfig().WithAppName("BubbleTea TUI")); err != nil {
		fmt.Fprintf(os.Stderr, "macgo: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

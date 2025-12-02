package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea" // Import Bubble Tea
)

// struct holds the "state" of application
type model struct {
	choice   []string // List of items in the menu
	cursor   int      // Which item is currently highlighted
	selected string   // Which item did the user choose
}

// initialModel define the starting state
func initialModel() model {
	return model{
		choice:   []string{"Download video", "Download Audio Only", "Exit"},
		cursor:   0, // Default top selected
		selected: "",
	}
}

// Init
func (m model) Init() tea.Cmd {
	return nil
}

// Update
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		// Quit keys
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choice)-1 {
				m.cursor++
			}

		case "enter":
			m.selected = m.choice[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

// View
func (m model) View() string {
	if m.selected != "" {
		return fmt.Sprintf("\nYou selected: %s\nGood choice!\n", m.selected)
	}

	// Header
	s := "\n What would you like to do?\n\n"

	// Loop through choice and render them
	for i, choice := range m.choice {
		cursor := "" // Default empty
		if m.cursor == i {
			cursor = ">" // Highlighting the current row
		}

		// rendering the row
		s += fmt.Sprintf("%s %s \n", cursor, choice)
	}

	// Footer (help text)
	s += "\n (Use arrow keys to move, Enter to select)\n"
	return s
}

func main() {

	// Start Bubble tea
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

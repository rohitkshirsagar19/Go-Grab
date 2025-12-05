package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput" // text input
	tea "github.com/charmbracelet/bubbletea"     // Import Bubble Tea
)

// int to represent screen states
const (
	StateMenu  = iota // 0
	StateInput = 1    // text input
	StateDone  = 2    // Download screen
)

// Model
type model struct {
	state      int      // screen
	choices    []string // menu items
	cursor     int      // menu cursor position
	choice     string   // video/audio
	textInput  textinput.Model
	enteredURL string // URL
}

func initialModel() model {
	// Initialize the text input component
	ti := textinput.New()
	ti.Placeholder = "Kripaya URL yaha daale...!"
	ti.Focus()
	ti.CharLimit = 160
	ti.Width = 40
	return model{
		state:     StateMenu,
		choices:   []string{"Download Video(mp4)", "Download Audio (mp3)", "Exit"},
		textInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink // blink cursor
}

// update
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// state management logic
	switch m.state {

	// STATE 0: THE MENU
	case StateMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "enter":
				// save the choice and move to the nxt state
				m.choice = m.choices[m.cursor]
				if m.choice == "Exit" {
					return m, tea.Quit
				}
				m.state = StateInput
			}
		}
	case StateInput:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				// save url
				m.enteredURL = m.textInput.Value()
				m.state = StateDone
				return m, tea.Quit
			}
		}

		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	}
	return m, nil
}

func (m model) View() string {
	switch m.state {
	case StateMenu:
		s := "\n GO-GRAB MEDIA DOWNLOADER \n\n"
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s += fmt.Sprintf(" %s %s \n", cursor, choice)
		}
		s += "\n (Select an option) \n"
		return s
	case StateInput:
		return fmt.Sprintf(
			"\n You selected %s \n\n%s \n (Esc to quit)\n",
			m.choice,
			m.textInput.View(),
		)
	case StateDone:
		return fmt.Sprintf("\n Ready to download! \n Mode: %s\n URL: %s \n", m.choice, m.enteredURL)
	}
	return ""
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

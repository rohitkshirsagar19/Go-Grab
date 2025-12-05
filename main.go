package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// States
const (
	StateMenu        = iota // 0
	StateInput              // 1
	StateDownloading        // 2
	StateDone               // 3
)

type progressMsg float64 // Message sent when download % changes
type doneMsg string      // Message sent when download finishes
type progressTickMsg struct{}

// model
type model struct {
	state        int
	choices      []string
	cursor       int
	choice       string
	textInput    textinput.Model
	enteredURL   string
	progress     progress.Model
	progressChan chan float64
	percent      float64 // Store current progress percentage
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Paste link here ..! (YouTube,X,Reddit,Insta)"
	ti.Focus()
	ti.CharLimit = 160
	ti.Width = 40

	// Initialize Progress Bar with a nice gradient
	prog := progress.New(progress.WithDefaultGradient())

	return model{
		state:        StateMenu,
		choices:      []string{"Download Video (MP4)", "Download Audio (MP3)", "Exit"},
		textInput:    ti,
		progress:     prog,
		progressChan: make(chan float64),
		percent:      0.0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	// Global Key Handling
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// If we are in the "Done" state, any key quits
		if m.state == StateDone {
			return m, tea.Quit
		}

	// Handle Progress Updates from background thread
	case progressMsg:
		if m.state == StateDownloading {
			// Store the percentage
			m.percent = float64(msg)

			return m, tea.Batch(
				waitForProgress(m.progressChan),
				m.progress.SetPercent(m.percent),
			)
		}

	// Handle Done Message
	case doneMsg:
		m.state = StateDone
		m.percent = 1.0
		return m, nil // Don't quit yet, let user see the Done screen

	// Handle progress tick for animation
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	switch m.state {
	case StateMenu:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "enter":
				m.choice = m.choices[m.cursor]
				if m.choice == "Exit" {
					return m, tea.Quit
				}
				m.state = StateInput
			}
		}

	case StateInput:
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			m.enteredURL = m.textInput.Value()
			m.state = StateDownloading
			m.percent = 0.0 // Reset progress

			// Start the download routine
			go downloadVideo(m.enteredURL, m.choice, m.progressChan)

			// Begin listening for progress
			return m, waitForProgress(m.progressChan)
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// view logic
func (m model) View() string {
	switch m.state {
	case StateMenu:
		s := "\n  GO-GRAB MEDIA DOWNLOADER\n\n"
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s += fmt.Sprintf("  %s %s\n", cursor, choice)
		}
		return s

	case StateInput:
		return fmt.Sprintf("\n  You selected: %s\n\n  %s\n", m.choice, m.textInput.View())

	case StateDownloading:
		return fmt.Sprintf("\n  Downloading from: %s\n\n  %s\n\n  Please wait...",
			m.enteredURL,
			m.progress.ViewAs(m.percent),
		)

	case StateDone:
		return "\n	 Done! File saved to current folder.\n  (Press any key to exit)\n"
	}
	return ""
}

func waitForProgress(c chan float64) tea.Cmd {
	return func() tea.Msg {
		// Wait for data from channel
		percent, ok := <-c
		if !ok {
			return doneMsg("Done")
		}
		return progressMsg(percent)
	}
}

func downloadVideo(url string, mode string, c chan float64) {
	defer close(c) // Close channel when finished to trigger doneMsg

	// Simplified Template: Just print the raw number
	args := []string{"--newline", "--progress-template", "%(progress._percent_str)s"}

	if mode == "Download Audio (MP3)" {
		args = append(args, "-x", "--audio-format", "mp3")
	} else {
		args = append(args, "-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best")
	}
	args = append(args, url)

	cmd := exec.Command("yt-dlp", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	if err := cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		line = strings.ReplaceAll(line, "%", "")

		// Attempt to parse whatever is left as a float
		// This ignores garbage lines like "[download]" and only catches numbers
		if p, err := strconv.ParseFloat(line, 64); err == nil {
			// Send percentage (0.0 - 1.0) to UI
			c <- p / 100
		}
	}
	cmd.Wait()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

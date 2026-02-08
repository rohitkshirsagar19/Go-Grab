package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// States
const (
	StateMenu        = iota // 0
	StateInput              // 1
	StateFetching           // 2
	StateQuality            // 3
	StateDownloading        // 4
	StateDone               // 5
	StateError              // 6
)

// Styles
type Styles struct {
	BorderColor lipgloss.Color
	InputField  lipgloss.Style
	Title       lipgloss.Style
	Info        lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	Spinner     lipgloss.Style
	Container   lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.BorderColor = lipgloss.Color("62")
	s.InputField = lipgloss.NewStyle().BorderForeground(s.BorderColor).BorderStyle(lipgloss.NormalBorder()).Padding(1).Width(60)
	s.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Padding(0, 1).Background(lipgloss.Color("62")).MarginBottom(1)
	s.Info = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	s.Error = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	s.Success = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	s.Spinner = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	s.Container = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(s.BorderColor).Padding(1, 2)
	return s
}

type progressMsg float64 // Message sent when download % changes
type doneMsg string      // Message sent when download finishes
type errMsg string       // Message sent when an error occurs
type metadataMsg VideoMetadata
type progressTickMsg struct{}

type VideoMetadata struct {
	Title      string `json:"title"`
	Uploader   string `json:"uploader"`
	Duration   int    `json:"duration"` // in seconds
	WebpageURL string `json:"webpage_url"`
}

type DownloadStats struct {
	ETA       string
	Speed     string
	TotalSize string
}

type statsMsg DownloadStats

// model
type model struct {
	state         int
	choices       []string
	cursor        int
	choice        string
	textInput     textinput.Model
	enteredURL    string
	progress      progress.Model
	progressChan  chan float64
	errChan       chan error
	statsChan     chan DownloadStats
	percent       float64
	stats         DownloadStats
	err           error
	spinner       spinner.Model
	styles        *Styles
	metadata      VideoMetadata
	width         int
	height        int
	qualities     []string
	qualityCursor int
}

func initialModel() model {
	s := DefaultStyles()

	ti := textinput.New()
	ti.Placeholder = "Paste link here ..! (YouTube,X,Reddit,Insta)"
	ti.Focus()
	ti.CharLimit = 160
	ti.Width = 50
	ti.Prompt = "🔗 "

	// Initialize Progress Bar with a nice gradient
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 50

	// Initialize Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = s.Spinner

	return model{
		state:        StateMenu,
		choices:      []string{"Download Video (MP4)", "Download Audio (MP3)", "Exit"},
		textInput:    ti,
		progress:     prog,
		progressChan: make(chan float64),
		errChan:      make(chan error),
		statsChan:    make(chan DownloadStats),
		percent:      0.0,
		spinner:      sp,
		styles:       s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
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
		// If we are in the "Done" or "Error" state, any key quits
		if m.state == StateDone || m.state == StateError {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles.Container.Width(min(msg.Width-4, 80)).Height(min(msg.Height-4, 20))

	// Handle Metadata Fetched
	case metadataMsg:
		m.metadata = VideoMetadata(msg)
		m.state = StateQuality
		m.qualities = []string{"Best Quality (MP4)", "1080p (MP4)", "720p (MP4)", "Audio Only (MP3)"}
		m.qualityCursor = 0
		return m, nil

	// Handle Progress Updates from background thread
	case progressMsg:
		if m.state == StateDownloading {
			// Store the percentage
			m.percent = float64(msg)

			return m, tea.Batch(
				waitForProgress(m.progressChan, m.statsChan, m.errChan),
				m.progress.SetPercent(m.percent),
			)
		}

	case statsMsg:
		m.stats = DownloadStats(msg)
		return m, waitForProgress(m.progressChan, m.statsChan, m.errChan)

	// Handle Spinner Tick
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	// Handle Done Message

	// Handle Done Message
	case doneMsg:
		m.state = StateDone
		m.percent = 1.0
		return m, nil

	// Handle Error Message
	case errMsg:
		m.state = StateError
		m.err = fmt.Errorf("%s", msg)
		return m, nil

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
			m.state = StateFetching
			return m, fetchMetadata(m.enteredURL)
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case StateQuality:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "up", "k":
				if m.qualityCursor > 0 {
					m.qualityCursor--
				}
			case "down", "j":
				if m.qualityCursor < len(m.qualities)-1 {
					m.qualityCursor++
				}
			case "enter":
				selectedQuality := m.qualities[m.qualityCursor]

				m.state = StateDownloading
				m.percent = 0.0 // Reset progress
				m.err = nil     // Reset error
				m.stats = DownloadStats{ETA: "...", Speed: "...", TotalSize: "..."}

				// Start the download routine
				go downloadVideo(m.metadata.WebpageURL, selectedQuality, m.progressChan, m.statsChan, m.errChan)

				// Begin listening for progress
				return m, waitForProgress(m.progressChan, m.statsChan, m.errChan)
			}
		}
	}

	return m, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// view logic
func (m model) View() string {
	var content string

	switch m.state {
	case StateMenu:
		content = m.styles.Title.Render("GO-GRAB MEDIA DOWNLOADER") + "\n\n"
		for i, choice := range m.choices {
			cursor := " "
			itemStyle := lipgloss.NewStyle().PaddingLeft(2)
			if m.cursor == i {
				cursor = ">"
				itemStyle = itemStyle.Foreground(lipgloss.Color("205")).Bold(true)
			}
			content += itemStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		}

	case StateInput:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			m.styles.Title.Render("ENTER URL"),
			m.styles.Info.Render(fmt.Sprintf("Mode: %s", m.choice)),
			m.textInput.View(),
		)

	case StateFetching:
		content = fmt.Sprintf("\n%s Fetching video information...", m.spinner.View())

	case StateQuality:
		content = m.styles.Title.Render("SELECT QUALITY") + "\n\n"
		content += m.styles.Info.Render(fmt.Sprintf("Video: %s", m.metadata.Title)) + "\n"
		content += m.styles.Info.Render(fmt.Sprintf("Uploader: %s", m.metadata.Uploader)) + "\n\n"

		for i, q := range m.qualities {
			cursor := " "
			itemStyle := lipgloss.NewStyle().PaddingLeft(2)
			if m.qualityCursor == i {
				cursor = ">"
				itemStyle = itemStyle.Foreground(lipgloss.Color("205")).Bold(true)
			}
			content += itemStyle.Render(fmt.Sprintf("%s %s", cursor, q)) + "\n"
		}

	case StateDownloading:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n\n%s\n\n%s  |  %s  |  %s",
			m.styles.Title.Render("DOWNLOADING"),
			m.styles.Info.Render(fmt.Sprintf("Source: %s", m.metadata.Title)),
			m.progress.ViewAs(m.percent),
			m.spinner.View()+" Downloading...",
			m.styles.Info.Render("ETA: "+m.stats.ETA),
			m.styles.Info.Render("Speed: "+m.stats.Speed),
			m.styles.Info.Render("Size: "+m.stats.TotalSize),
		)

	case StateDone:
		content = m.styles.Success.Render("\n  ✨ Done! File saved to current folder.\n  (Press any key to exit)\n")

	case StateError:
		content = m.styles.Error.Render(fmt.Sprintf("\n  ❌ Error: %v\n  (Press any key to exit)\n", m.err))
	}

	// Wrap content in a container
	return m.styles.Container.Render(content)
}

func waitForProgress(progChan chan float64, statsChan chan DownloadStats, errChan chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case percent, ok := <-progChan:
			if !ok {
				return doneMsg("Done")
			}
			return progressMsg(percent)
		case stats := <-statsChan:
			return statsMsg(stats)
		case err := <-errChan:
			return errMsg(err.Error())
		}
	}
}

func downloadVideo(url string, mode string, progChan chan float64, statsChan chan DownloadStats, errChan chan error) {
	defer close(progChan)
	// defer close(errChan)

	// Template to get: [percent] [FPS] [ETA] [Speed] [TotalSize]
	// We use a custom separator "|" to parse it easily
	// Note: yt-dlp python string formatting
	template := "%(progress._percent_str)s|%(progress._eta_str)s|%(progress._speed_str)s|%(progress._total_bytes_estimate_str)s"

	args := []string{"--newline", "--progress-template", template}

	if mode == "Audio Only (MP3)" || mode == "Download Audio (MP3)" {
		args = append(args, "-x", "--audio-format", "mp3")
	} else if mode == "1080p (MP4)" {
		args = append(args, "-f", "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[height<=1080][ext=mp4]/best")
	} else if mode == "720p (MP4)" {
		args = append(args, "-f", "bestvideo[height<=720][ext=mp4]+bestaudio[ext=m4a]/best[height<=720][ext=mp4]/best")
	} else {
		// Best Quality
		args = append(args, "-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best")
	}
	args = append(args, url)

	cmdName := getYoutubeDLCommand()
	cmd := exec.Command(cmdName, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errChan <- err
		return
	}

	// Capture stderr to see errors
	stderr, err := cmd.StderrPipe()
	if err != nil {
		errChan <- err
		return
	}

	if err := cmd.Start(); err != nil {
		errChan <- err
		return
	}

	// Read stderr in a separate goroutine to avoid blocking
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			// You could stream stderr logs here if you had a debug view
			// For now, we'll just ignore it until the end or if we want to parse it for specific errors
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, "|")

		if len(parts) >= 1 {
			// Part 0 is percent
			pStr := strings.ReplaceAll(parts[0], "%", "")
			if p, err := strconv.ParseFloat(pStr, 64); err == nil {
				progChan <- p / 100
			}
		}

		if len(parts) >= 4 {
			// parts[1] = ETA, parts[2] = Speed, parts[3] = TotalSize
			stats := DownloadStats{
				ETA:       parts[1],
				Speed:     parts[2],
				TotalSize: parts[3],
			}
			statsChan <- stats
		}
	}

	if err := cmd.Wait(); err != nil {
		errChan <- fmt.Errorf("download failed: %v", err)
	}
}

// Helper to find yt-dlp
// Priority:
// 1. Next to the executable (useful when installed globally)
// 2. In current directory (useful for dev)
// 3. In PATH (system default)
func getYoutubeDLCommand() string {
	// Check next to executable
	if ex, err := os.Executable(); err == nil {
		localPath := filepath.Join(filepath.Dir(ex), "yt-dlp")
		if _, err := os.Stat(localPath); err == nil {
			return localPath
		}
	}

	// Check current directory
	if _, err := os.Stat("./yt-dlp"); err == nil {
		return "./yt-dlp"
	}

	// Default to PATH
	return "yt-dlp"
}

func fetchMetadata(url string) tea.Cmd {
	return func() tea.Msg {
		cmdName := getYoutubeDLCommand()

		cmd := exec.Command(cmdName, "--dump-json", url)
		output, err := cmd.Output()
		if err != nil {
			return errMsg(fmt.Sprintf("Failed to fetch metadata: %v", err))
		}

		var meta VideoMetadata
		if err := json.Unmarshal(output, &meta); err != nil {
			return errMsg(fmt.Sprintf("Failed to parse metadata: %v", err))
		}

		// Ensure URL is preserved if not in JSON (unlikely but good safety)
		if meta.WebpageURL == "" {
			meta.WebpageURL = url
		}

		return metadataMsg(meta)
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

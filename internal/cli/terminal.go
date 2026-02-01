package cli

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
)

// Spinner frames for animated progress
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Terminal provides terminal-aware output utilities
type Terminal struct {
	IsTerminal   bool
	UseColor     bool
	spinnerIndex int
}

// NewTerminal creates a new Terminal instance
func NewTerminal() *Terminal {
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	return &Terminal{
		IsTerminal: isTerminal,
		UseColor:   isTerminal, // Only use color in terminal
	}
}

// ClearLine clears the current line (terminal only)
func (t *Terminal) ClearLine() {
	if t.IsTerminal {
		fmt.Print("\r\033[K")
	}
}

// Flush ensures output is written immediately
func (t *Terminal) Flush() {
	os.Stdout.Sync()
}

// Spinner returns the next spinner frame
func (t *Terminal) Spinner() string {
	if !t.IsTerminal {
		return ""
	}
	frame := spinnerFrames[t.spinnerIndex]
	t.spinnerIndex = (t.spinnerIndex + 1) % len(spinnerFrames)
	return frame
}

// Color wraps text in ANSI color codes (terminal only)
func (t *Terminal) Color(color, text string) string {
	if !t.UseColor {
		return text
	}
	return color + text + ColorReset
}

// FormatETA formats a duration as a human-readable ETA string
func FormatETA(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s > 0 {
			return fmt.Sprintf("%dm%ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// PhaseColor returns the appropriate color for a sync phase
func PhaseColor(phase string) string {
	switch phase {
	case "listing":
		return ColorCyan
	case "fetching":
		return ColorBlue
	case "filtering":
		return ColorYellow
	case "classifying":
		return ColorPurple
	case "validating":
		return ColorPurple
	case "processing":
		return ColorGreen
	case "updating_status":
		return ColorGray
	default:
		return ColorWhite
	}
}

package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// SimpleProgress represents a simple progress bar with mutex
type Progress struct {
	sync.Mutex
	total       int
	current     int
	lastWidth   int
	width       int
	startTime   time.Time
	lastUpdate  time.Time
	completed   bool
	description string
}

// New creates a new progress bar
func New(total int) *Progress {
	return &Progress{
		total:      total,
		current:    0,
		lastWidth:  0,
		width:      40, // Fixed width of 40 characters
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

// SetDescription sets the description text for the progress bar
func (p *Progress) SetDescription(desc string) {
	p.Lock()
	defer p.Unlock()
	p.description = desc
}

// Update updates the progress bar with the current progress
func (p *Progress) Update(current int) {
	p.Lock()
	defer p.Unlock()

	// Don't update too often to avoid flickering (max 10 updates per second)
	if time.Since(p.lastUpdate) < 100*time.Millisecond && current < p.total && !p.completed {
		return
	}

	p.current = current
	p.lastUpdate = time.Now()

	if p.current >= p.total {
		p.completed = true
	}

	// Calculate percentage
	percent := 0.0
	if p.total > 0 {
		percent = float64(p.current) / float64(p.total) * 100
	}

	filledWidth := 0
	if p.total > 0 {
		filledWidth = int(float64(p.width) * float64(p.current) / float64(p.total))
	}
	if filledWidth < 0 {
		filledWidth = 0
	}
	if filledWidth > p.width {
		filledWidth = p.width
	}

	// Create the progress bar
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", p.width-filledWidth)

	// Calculate ETA
	var eta string
	if p.current > 0 && p.total > 0 && p.current < p.total {
		elapsed := time.Since(p.startTime)
		estimatedTotal := elapsed * time.Duration(p.total) / time.Duration(p.current)
		remaining := estimatedTotal - elapsed
		if remaining < 0 {
			remaining = 0
		}
		if remaining < 100*time.Hour {
			eta = fmt.Sprintf("ETA: %s", formatDuration(remaining))
		} else {
			eta = "ETA: --:--"
		}
	} else if p.completed {
		elapsed := time.Since(p.startTime)
		eta = fmt.Sprintf("Time: %s", formatDuration(elapsed))
	} else {
		eta = "ETA: --:--"
	}

	// Status line
	var statusLine string
	if p.description != "" {
		statusLine = fmt.Sprintf("%s: [%s] %.1f%% (%d/%d) %s",
			p.description, bar, percent, p.current, p.total, eta)
	} else {
		statusLine = fmt.Sprintf("[%s] %.1f%% (%d/%d) %s",
			bar, percent, p.current, p.total, eta)
	}

	// Clear previous line if it was longer
	if len(statusLine) < p.lastWidth {
		fmt.Print("\r" + strings.Repeat(" ", p.lastWidth) + "\r")
	} else {
		fmt.Print("\r")
	}
	p.lastWidth = len(statusLine)

	// Print status line
	if p.completed {
		// Green for completed and add a newline
		fmt.Print("\r") // Ensure we're at the beginning of the line
		color.Green(statusLine)
		fmt.Println() // Add explicit newline after completion
	} else {
		// Regular output for in progress (no newline)
		fmt.Print(statusLine)
	}
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	hours := d / time.Hour
	d -= hours * time.Hour

	minutes := d / time.Minute
	d -= minutes * time.Minute

	seconds := d / time.Second

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

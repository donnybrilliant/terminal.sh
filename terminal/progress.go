package terminal

import (
	"fmt"
	"time"
)

// ProgressBar handles progress bar rendering
type ProgressBar struct {
	width int
}

// NewProgressBar creates a new progress bar
func NewProgressBar() *ProgressBar {
	return &ProgressBar{
		width: 50, // Default width
	}
}

// Show displays a progress bar for an operation
func (p *ProgressBar) Show(message string, duration time.Duration, updateCallback func(float64)) {
	startTime := time.Now()
	ticker := time.NewTicker(50 * time.Millisecond) // Update every 50ms
	defer ticker.Stop()

	for {
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(duration)
		
		if progress > 1.0 {
			progress = 1.0
		}

		// Call update callback if provided
		if updateCallback != nil {
			updateCallback(progress)
		}

		// Render progress bar
		p.Render(message, progress)

		if progress >= 1.0 {
			break
		}

		<-ticker.C
	}

	// Final render at 100%
	p.Render(message, 1.0)
	fmt.Println() // New line after progress
}

// Render renders a progress bar
func (p *ProgressBar) Render(message string, progress float64) {
	if progress > 1.0 {
		progress = 1.0
	}
	if progress < 0 {
		progress = 0
	}

	filled := int(float64(p.width) * progress)
	empty := p.width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}

	percentage := int(progress * 100)
	
	// Use carriage return to overwrite the line
	fmt.Printf("\r%s [%s] %d%%", message, bar, percentage)
}

// RenderSimple renders a simple progress bar without message
func (p *ProgressBar) RenderSimple(progress float64) {
	p.Render("", progress)
}


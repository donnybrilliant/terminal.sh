package cmd

import (
	"fmt"
	"time"
)

// showProgressBar displays a progress bar for an operation (inlined to avoid import cycle)
func showProgressBar(message string, duration time.Duration) {
	startTime := time.Now()
	ticker := time.NewTicker(50 * time.Millisecond) // Update every 50ms
	defer ticker.Stop()

	for {
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(duration)
		
		if progress > 1.0 {
			progress = 1.0
		}

		// Render progress bar
		renderProgressBar(message, progress)

		if progress >= 1.0 {
			break
		}

		<-ticker.C
	}

	// Final render at 100%
	renderProgressBar(message, 1.0)
	fmt.Println() // New line after progress
}

// renderProgressBar renders a progress bar
func renderProgressBar(message string, progress float64) {
	if progress > 1.0 {
		progress = 1.0
	}
	if progress < 0 {
		progress = 0
	}

	width := 50
	filled := int(float64(width) * progress)
	empty := width - filled

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


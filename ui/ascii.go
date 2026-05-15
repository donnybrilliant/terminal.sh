package ui

import (
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Animation constants
const (
	ASCIIGradientFrameCount = 18
	ASCIIGradientFrameDelay = 75 * time.Millisecond
	ASCIIGradientWidthCap   = 180
	ASCIIGradientHeightCap  = 80
)

// ASCIIAnimationState represents the state of an ASCII art animation
type ASCIIAnimationState struct {
	// Animation state
	gradientFrames    []string
	gradientFrameIdx  int
	gradientAnimating bool
	gradientSeed      int64
	
	// ASCII art to display
	asciiArt      string
	asciiArtWidth int
	asciiArtHeight int
	
	// Screen dimensions
	width  int
	height int
	
	// Animation phase: "gradient" -> "ascii" -> "complete"
	phase string
	
	// Frame counters for auto-completion
	asciiPhaseFrameCount int // Count frames in ASCII phase
}

// NewASCIIAnimation creates a new ASCII animation state
func NewASCIIAnimation(text string, width, height int, colors []string, charWidth, charHeight int) *ASCIIAnimationState {
	// Generate ASCII art from text
	asciiArt := StringToASCIIArt(text, charWidth, charHeight)
	
	// Render with gradient colors
	styledArt := RenderASCIIArtWithGradient(asciiArt, 0, colors) // Don't stretch yet, we'll center it
	
	// Calculate ASCII art dimensions
	lines := strings.Split(strings.TrimPrefix(styledArt, "\n"), "\n")
	asciiWidth := 0
	for _, line := range lines {
		if len([]rune(line)) > asciiWidth {
			asciiWidth = len([]rune(line))
		}
	}
	asciiHeight := len(lines)
	
	return &ASCIIAnimationState{
		gradientAnimating: true,
		gradientSeed:      time.Now().UnixNano(),
		gradientFrameIdx:  0, // Initialize to 0
		asciiArt:          styledArt,
		asciiArtWidth:     asciiWidth,
		asciiArtHeight:    asciiHeight,
		width:             width,
		height:            height,
		phase:             "gradient",
	}
}

// BuildGradientFrames builds the gradient animation frames
func (a *ASCIIAnimationState) BuildGradientFrames() {
	width := a.width
	height := a.height

	// Ensure reasonable bounds
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	// Don't cap width - use full terminal width for proper stretching
	// Only cap height to prevent excessive rendering
	if height > ASCIIGradientHeightCap {
		height = ASCIIGradientHeightCap
	}

	usableHeight := height - 1 // reserve one line
	if usableHeight < 4 {
		usableHeight = 4
	}

	// Palette: magenta gradient plus monochrome tones
	primary := []string{"205", "213", "207", "219", "218", "212", "205"}
	greys := []string{"232", "235", "237", "240", "244", "248", "252", "255"}
	palette := append(primary, greys...)

	rng := rand.New(rand.NewSource(a.gradientSeed))
	frames := make([]string, ASCIIGradientFrameCount)

	for f := 0; f < ASCIIGradientFrameCount; f++ {
		var sb strings.Builder
		phase := rng.Float64() * 6

		for y := 0; y < usableHeight; y++ {
			for x := 0; x < width; x++ {
				// Wave + noise to keep the gradient organic
				wave := math.Sin((float64(x)+phase)/5.0) + math.Cos((float64(y)+float64(f)*1.4)/4.0)
				baseIdx := int(math.Abs(wave) * float64(len(palette)))
				baseIdx = clampInt(baseIdx, 0, len(palette)-1)

				// Occasionally inject bright accent pixels
				if (x+y+f)%13 == 0 || rng.Float64() > 0.92 {
					accentIdx := int(math.Abs(math.Sin(float64(f)+float64(x)/3+float64(y)/3)) * float64(len(primary)))
					accentIdx = clampInt(accentIdx, 0, len(primary)-1)
					sb.WriteString("\x1b[38;5;")
					sb.WriteString(primary[accentIdx])
					sb.WriteString("m█")
					continue
				}

				sb.WriteString("\x1b[38;5;")
				sb.WriteString(palette[baseIdx%len(palette)])
				sb.WriteString("m█")
			}

			// Reset color at end of each line
			sb.WriteString("\x1b[0m")
			if y < usableHeight-1 {
				sb.WriteString("\n")
			}
		}

		frames[f] = sb.String()
	}

	a.gradientFrames = frames
}

// GetCurrentFrame returns the current animation frame
func (a *ASCIIAnimationState) GetCurrentFrame() string {
	if a.phase == "complete" {
		return ""
	}
	
	// Always ensure gradient frames are built
	if len(a.gradientFrames) == 0 {
		a.BuildGradientFrames()
	}
	
	// Get current gradient frame (cycling continuously)
	if len(a.gradientFrames) == 0 {
		return "" // No frames available
	}
	gradientFrame := a.gradientFrames[a.gradientFrameIdx%len(a.gradientFrames)]
	
	// If in ASCII phase, overlay ASCII art on top of gradient
	if a.phase == "ascii" {
		return a.getASCIIFrameOverlay(gradientFrame)
	}
	
	// In gradient phase, just show gradient
	return gradientFrame
}

// getASCIIFrameOverlay overlays ASCII art on top of gradient background
func (a *ASCIIAnimationState) getASCIIFrameOverlay(gradientFrame string) string {
	// Split gradient into lines
	gradientLines := strings.Split(gradientFrame, "\n")
	
	// Get ASCII art lines (raw, without centering - we'll center manually)
	asciiRawLines := strings.Split(strings.TrimPrefix(a.asciiArt, "\n"), "\n")
	
	// Calculate vertical center position
	gradientHeight := len(gradientLines)
	asciiHeight := len(asciiRawLines)
	startRow := (gradientHeight - asciiHeight) / 2
	if startRow < 0 {
		startRow = 0
	}
	
	// Build result by overlaying ASCII on gradient
	var result strings.Builder
	for i, gradientLine := range gradientLines {
		if i >= startRow && i < startRow+asciiHeight {
			// This line should have ASCII art overlay
			asciiIdx := i - startRow
			if asciiIdx < len(asciiRawLines) {
				asciiLine := asciiRawLines[asciiIdx]
				// Overlay ASCII on gradient line
				result.WriteString(a.overlayLine(gradientLine, asciiLine, a.width))
			} else {
				result.WriteString(gradientLine)
			}
		} else {
			// Just gradient
			result.WriteString(gradientLine)
		}
		if i < len(gradientLines)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// overlayLine overlays ASCII line on gradient line, centering ASCII horizontally
// Simple approach: extract gradient color, write padding, write ASCII, write more padding
func (a *ASCIIAnimationState) overlayLine(gradientLine, asciiLine string, width int) string {
	// Strip ANSI codes to get visible length for positioning
	visibleASCII := a.stripANSI(asciiLine)
	visibleLen := len([]rune(visibleASCII))
	
	if visibleLen == 0 || strings.TrimSpace(visibleASCII) == "" {
		// No ASCII content, use gradient
		return gradientLine
	}
	
	// Extract the gradient color code (first ANSI code in the line)
	gradientColor := a.extractFirstANSICode(gradientLine)
	if gradientColor == "" {
		gradientColor = "\x1b[38;5;205m" // Default magenta
	}
	
	// Calculate horizontal center position
	leftPad := (width - visibleLen) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	
	var result strings.Builder
	
	// Write left padding with gradient color
	result.WriteString(gradientColor)
	for i := 0; i < leftPad; i++ {
		result.WriteString("█")
	}
	
	// Write ASCII art (with its own color codes - they'll override the gradient)
	result.WriteString(asciiLine)
	
	// Write right padding
	remaining := width - leftPad - visibleLen
	if remaining > 0 {
		result.WriteString(gradientColor)
		for i := 0; i < remaining; i++ {
			result.WriteString("█")
		}
	}
	
	// Reset at end
	result.WriteString("\x1b[0m")
	
	return result.String()
}

// extractFirstANSICode extracts the first ANSI color code from a string
func (a *ASCIIAnimationState) extractFirstANSICode(s string) string {
	idx := strings.Index(s, "\x1b[")
	if idx == -1 {
		return ""
	}
	// Find the end of the ANSI code (the 'm')
	endIdx := strings.Index(s[idx:], "m")
	if endIdx == -1 {
		return ""
	}
	return s[idx : idx+endIdx+1]
}

// stripANSI removes ANSI escape codes from a string to get visible length
func (a *ASCIIAnimationState) stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// AdvanceFrame advances to the next animation frame
func (a *ASCIIAnimationState) AdvanceFrame() {
	// Always ensure gradient frames are built
	if len(a.gradientFrames) == 0 {
		a.BuildGradientFrames()
	}
	
	if a.phase == "gradient" {
		// Advance gradient frame
		a.gradientFrameIdx++
		// After a few gradient frames, transition to ASCII overlay phase
		if a.gradientFrameIdx >= 3 {
			a.phase = "ascii"
			a.asciiPhaseFrameCount = 0 // Reset counter when entering ASCII phase
		}
	} else if a.phase == "ascii" {
		// Continue cycling gradient frames in background while ASCII is shown
		// This creates the animated background effect
		a.gradientFrameIdx = (a.gradientFrameIdx + 1) % len(a.gradientFrames)
		// Auto-complete after ~2 seconds (about 27 frames at 75ms per frame)
		a.asciiPhaseFrameCount++
		if a.asciiPhaseFrameCount >= 27 {
			// Auto-complete if timer didn't fire
			a.phase = "complete"
			a.gradientAnimating = false
		}
	}
}

// IsComplete returns true if the animation is complete
func (a *ASCIIAnimationState) IsComplete() bool {
	return a.phase == "complete"
}

// Complete marks the animation as complete (falls away)
func (a *ASCIIAnimationState) Complete() {
	a.phase = "complete"
	a.gradientAnimating = false
}

// IsAnimating returns true if animation is still running
func (a *ASCIIAnimationState) IsAnimating() bool {
	return a.phase != "complete"
}

// GetPhase returns the current animation phase
func (a *ASCIIAnimationState) GetPhase() string {
	return a.phase
}

// UpdateDimensions updates the screen dimensions and rebuilds frames
func (a *ASCIIAnimationState) UpdateDimensions(width, height int) {
	a.width = width
	a.height = height
	a.BuildGradientFrames()
}

// clampInt clamps a value between min and max
func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// StringToAnimatedASCIIArt creates a full-screen animated ASCII art sequence:
// 1. Gradient animation fills the screen
// 2. ASCII art appears centered
// 3. Everything "falls away" (disappears)
//
// This returns an ASCIIAnimationState that manages the entire animation sequence.
//
// Parameters:
//   - text: The string to convert to ASCII art
//   - width: Screen width
//   - height: Screen height
//   - colors: Optional color palette (nil uses default magenta/pink gradient)
//   - charWidth: Width of each character in the ASCII art (default 6 if <= 0)
//   - charHeight: Height of each character in the ASCII art (default 7 if <= 0)
//
// Usage:
//   anim := StringToAnimatedASCIIArt("WELCOME", 80, 24, nil, 6, 7)
//   frame := anim.GetCurrentFrame() // Get current frame to display
//   anim.AdvanceFrame() // Advance to next frame
//   if anim.IsComplete() { /* animation done */ }
func StringToAnimatedASCIIArt(text string, width, height int, colors []string, charWidth, charHeight int) *ASCIIAnimationState {
	return NewASCIIAnimation(text, width, height, colors, charWidth, charHeight)
}

// StretchASCIIArt stretches ASCII art to fit a target width while preserving visual structure
func StretchASCIIArt(asciiArt string, targetWidth int) string {
	if targetWidth <= 0 {
		return asciiArt
	}
	
	lines := strings.Split(strings.TrimPrefix(asciiArt, "\n"), "\n")
	if len(lines) == 0 {
		return asciiArt
	}
	
	// Find the maximum line width in the original art
	maxWidth := 0
	for _, line := range lines {
		lineWidth := len([]rune(line))
		if lineWidth > maxWidth {
			maxWidth = lineWidth
		}
	}
	
	if maxWidth == 0 {
		return asciiArt
	}
	
	// If target width is smaller or equal, return original
	if targetWidth <= maxWidth {
		return asciiArt
	}
	
	// Calculate stretch factor
	stretchFactor := float64(targetWidth) / float64(maxWidth)
	
	var stretched strings.Builder
	for _, line := range lines {
		if line == "" {
			stretched.WriteString("\n")
			continue
		}
		
		// Stretch the line using rune-aware stretching
		stretchedLine := stretchLineRunes(line, targetWidth, stretchFactor)
		stretched.WriteString(stretchedLine)
		stretched.WriteString("\n")
	}
	
	return strings.TrimSuffix(stretched.String(), "\n")
}

// stretchLineRunes stretches a single line to target width using rune-aware stretching
func stretchLineRunes(line string, targetWidth int, stretchFactor float64) string {
	runes := []rune(line)
	if len(runes) == 0 {
		return strings.Repeat(" ", targetWidth)
	}
	
	var result strings.Builder
	lineLen := len(runes)
	
	for i := 0; i < targetWidth; i++ {
		sourcePos := float64(i) / stretchFactor
		sourceIdx := int(sourcePos)
		
		if sourceIdx >= lineLen {
			sourceIdx = lineLen - 1
		}
		
		result.WriteRune(runes[sourceIdx])
	}
	
	return result.String()
}

// RenderASCIIArtWithGradient renders ASCII art with gradient colors and optional width stretching
func RenderASCIIArtWithGradient(asciiArt string, width int, colors []string) string {
	// Stretch if width is specified
	if width > 0 {
		asciiArt = StretchASCIIArt(asciiArt, width)
	}
	
	// Default color gradient if none provided
	if len(colors) == 0 {
		colors = []string{"205", "213", "207", "219", "218", "212", "205"} // Magenta/pink gradient
	}
	
	var styled strings.Builder
	lines := strings.Split(strings.TrimPrefix(asciiArt, "\n"), "\n")
	
	for lineIdx, line := range lines {
		if line == "" {
			styled.WriteString("\n")
			continue
		}
		// Cycle through colors for each line
		color := colors[lineIdx%len(colors)]
		lineStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Bold(true)
		styled.WriteString(lineStyle.Render(line))
		styled.WriteString("\n")
	}
	
	return strings.TrimSuffix(styled.String(), "\n")
}

// StringToASCIIArt converts a string to block-style ASCII art using Unicode block characters.
func StringToASCIIArt(text string, charWidth, charHeight int) string {
	if text == "" {
		return ""
	}
	
	// Default dimensions
	if charWidth <= 0 {
		charWidth = 6
	}
	if charHeight <= 0 {
		charHeight = 7
	}
	
	// Convert text to uppercase for better ASCII art rendering
	text = strings.ToUpper(text)
	
	// Character patterns for block-style ASCII art
	charPatterns := getASCIICharPatterns()
	
	// Build ASCII art line by line
	var result strings.Builder
	lines := make([]strings.Builder, charHeight)
	
	for _, char := range text {
		pattern, exists := charPatterns[char]
		if !exists {
			// Use space pattern for unknown characters
			pattern = charPatterns[' ']
		}
		
		// Split pattern into lines
		patternLines := strings.Split(pattern, "\n")
		if len(patternLines) == 0 {
			continue
		}
		
		// Get actual pattern dimensions
		patternHeight := len(patternLines)
		patternWidth := 0
		for _, line := range patternLines {
			lineWidth := len([]rune(line))
			if lineWidth > patternWidth {
				patternWidth = lineWidth
			}
		}
		if patternWidth == 0 {
			patternWidth = 5 // Default fallback
		}
		
		// Calculate scaling factors - simple integer multipliers
		// Since we use integer multipliers, these should be whole numbers
		scaleX := charWidth / patternWidth
		scaleY := charHeight / patternHeight
		
		// Ensure minimum scale of 1
		if scaleX < 1 {
			scaleX = 1
		}
		if scaleY < 1 {
			scaleY = 1
		}
		
		// Generate scaled character by duplicating blocks both horizontally and vertically
		// With integer multipliers, each pattern line gets duplicated scaleY times
		
		// Now generate the scaled output
		outputLineIdx := 0
		for _, patternLine := range patternLines {
			if outputLineIdx >= charHeight {
				break
			}
			
			// Scale the pattern line horizontally by duplicating each character
			patternRunes := []rune(patternLine)
			var scaledLine strings.Builder
			for _, r := range patternRunes {
				// Duplicate each character scaleX times
				scaledLine.WriteString(strings.Repeat(string(r), scaleX))
			}
			
			scaledLineStr := scaledLine.String()
			scaledRunes := []rune(scaledLineStr)
			
			// Trim or pad to exact charWidth
			if len(scaledRunes) > charWidth {
				scaledLineStr = string(scaledRunes[:charWidth])
			} else if len(scaledRunes) < charWidth {
				// Pad with spaces
				for len(scaledRunes) < charWidth {
					scaledLineStr += " "
					scaledRunes = []rune(scaledLineStr)
				}
			}
			
			// Output this scaled line scaleY times (vertical duplication)
			for i := 0; i < scaleY && outputLineIdx < charHeight; i++ {
				lines[outputLineIdx].WriteString(scaledLineStr)
				outputLineIdx++
			}
		}
		
		// Fill any remaining lines with spaces (shouldn't happen with integer multipliers, but safety check)
		for outputLineIdx < charHeight {
			lines[outputLineIdx].WriteString(strings.Repeat(" ", charWidth))
			outputLineIdx++
		}
		
		// Add spacing between characters (scales with size)
		// Add 1-2 spaces scaled by the multiplier
		spacing := scaleX
		if spacing < 1 {
			spacing = 1
		}
		// Cap spacing at 3 to avoid too much space
		if spacing > 3 {
			spacing = 3
		}
		for i := 0; i < charHeight; i++ {
			lines[i].WriteString(strings.Repeat(" ", spacing))
		}
	}
	
	// Combine all lines
	for i, line := range lines {
		result.WriteString(line.String())
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// getASCIICharPatterns returns a map of character to ASCII art pattern
// Uses Unicode block characters (█) similar to the TERMINAL.SH style
func getASCIICharPatterns() map[rune]string {
	return map[rune]string{
		'A': `█████
█   █
█████
█   █
█   █
█   █
█   █`,
		'B': `█████
█   █
█████
█   █
█████
█   █
█████`,
		'C': `█████
█
█
█
█
█
█████`,
		'D': `████
█   █
█   █
█   █
█   █
█   █
████`,
		'E': `█████
█
█
█████
█
█
█████`,
		'F': `█████
█
█
█████
█
█
█`,
		'G': `█████
█
█
█  ██
█   █
█   █
█████`,
		'H': `█   █
█   █
█   █
█████
█   █
█   █
█   █`,
		'I': `█████
  █
  █
  █
  █
  █
█████`,
		'J': `█████
    █
    █
    █
    █
█   █
█████`,
		'K': `█   █
█  █
█ █
██
█ █
█  █
█   █`,
		'L': `█
█
█
█
█
█
█████`,
		'M': `█   █
██ ██
█ █ █
█   █
█   █
█   █
█   █`,
		'N': `█   █
██  █
█ █ █
█  ██
█   █
█   █
█   █`,
		'O': `█████
█   █
█   █
█   █
█   █
█   █
█████`,
		'P': `█████
█   █
█   █
█████
█
█
█`,
		'Q': `█████
█   █
█   █
█   █
█ █ █
█  █
████ █`,
		'R': `█████
█   █
█   █
█████
█  █
█   █
█   █`,
		'S': `█████
█
█
█████
    █
    █
█████`,
		'T': `█████
  █
  █
  █
  █
  █
  █`,
		'U': `█   █
█   █
█   █
█   █
█   █
█   █
█████`,
		'V': `█   █
█   █
█   █
█   █
█   █
 █ █
  █`,
		'W': `█   █
█   █
█   █
█   █
█ █ █
██ ██
█   █`,
		'X': `█   █
 █ █
  █
  █
  █
 █ █
█   █`,
		'Y': `█   █
█   █
 █ █
  █
  █
  █
  █`,
		'Z': `█████
    █
   █
  █
 █
█
█████`,
		'0': `█████
█   █
█   █
█   █
█   █
█   █
█████`,
		'1': `  █
 ██
  █
  █
  █
  █
█████`,
		'2': `█████
    █
    █
█████
█
█
█████`,
		'3': `█████
    █
    █
█████
    █
    █
█████`,
		'4': `█   █
█   █
█   █
█████
    █
    █
    █`,
		'5': `█████
█
█
█████
    █
    █
█████`,
		'6': `█████
█
█
█████
█   █
█   █
█████`,
		'7': `█████
    █
    █
    █
    █
    █
    █`,
		'8': `█████
█   █
█   █
█████
█   █
█   █
█████`,
		'9': `█████
█   █
█   █
█████
    █
    █
█████`,
		' ': `     
     
     
     
     
     
     `,
		'.': `
     
     
     
     
     
     
  █`,
		'!': `  █
  █
  █
  █
  █
     
  █`,
		'?': `█████
    █
    █
  ███
  █
     
  █`,
		':': `
     
  █
     
     
  █
     
     `,
		'-': `
     
     
     
█████
     
     
     `,
		'_': `
     
     
     
     
     
     
█████`,
		'/': `    █
   █
  █
 █
█
     
     `,
		'\\': `█
 █
  █
   █
    █
     
     `,
	}
}

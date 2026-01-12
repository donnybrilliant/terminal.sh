package ui

import (
	"fmt"
	"net"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds all color codes for the terminal theme
type Theme struct {
	// Primary colors
	Primary   string // 205 - Magenta
	Secondary string // 39 - Blue
	
	// Semantic colors
	Success string // 46 - Green
	Error   string // 196 - Red
	Warning string // 220 - Yellow/Orange
	Info    string // 51 - Cyan
	
	// Accent colors
	Pink    string // 213 - Pink
	LightPink string // 207 - Light Pink
	Cyan    string // 51 - Cyan
	LightBlue string // 45 - Light Blue
	Green   string // 46 - Green
	
	// Neutral colors
	LightGray string // 252 - Light Gray
	Gray      string // 240 - Gray
	DarkGray  string // 235 - Dark Gray
}

// DefaultTheme returns the default theme matching current color scheme
var DefaultTheme = Theme{
	Primary:   "205",
	Secondary: "39",
	Success:   "46",
	Error:     "196",
	Warning:   "220",
	Info:      "51",
	Pink:      "213",
	LightPink: "207",
	Cyan:      "51",
	LightBlue: "45",
	Green:     "46",
	LightGray: "252",
	Gray:      "240",
	DarkGray:  "235",
}

// Current theme (can be swapped in the future)
var currentTheme = DefaultTheme

// Exported style variables for use across the codebase
var (
	// Header styles
	HeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Primary)).
		Bold(true)
	
	SectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Secondary)).
		Bold(true)
	
	// Text styles
	LabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Info))
	
	ValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.LightGray))
	
	ListStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Secondary))
	
	// Semantic styles
	SuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Success)).
		Bold(true)
	
	SuccessStyleNoBold = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Success))
	
	InfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Info))
	
	WarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Warning)).
		Bold(true)
	
	ErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Error)).
		Bold(true)
	
	// Accent styles
	AccentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Pink))
	
	AccentBoldStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Pink)).
		Bold(true)
	
	PriceStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Success))
	
	// Gray styles
	GrayStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Gray))
	
	// Security level styles (for dynamic use)
	SecurityLowStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Success))
	
	SecurityMediumStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Warning))
	
	SecurityHighStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Error))
	
	// Command-specific styles
	CommandStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Success))
	
	ToolCommandStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Pink))
	
	FilesystemCommandStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Success))
)

// GetSecurityStyle returns the appropriate style for a security level
func GetSecurityStyle(level int) lipgloss.Style {
	if level <= 3 {
		return SecurityLowStyle
	} else if level <= 6 {
		return SecurityMediumStyle
	}
	return SecurityHighStyle
}

// FormatSectionHeader formats a section header with optional emoji
func FormatSectionHeader(title, emoji string) string {
	if emoji != "" {
		title = emoji + " " + title
	}
	return HeaderStyle.Render(title) + "\n"
}

// FormatKeyValuePair formats a key-value pair
func FormatKeyValuePair(key, value string) string {
	return LabelStyle.Render(key) + ": " + ValueStyle.Render(value)
}

// FormatListItem formats a list item with optional emoji
func FormatListItem(text, emoji string) string {
	prefix := "  "
	if emoji != "" {
		prefix = fmt.Sprintf("  %s ", emoji)
	}
	return ListStyle.Render(prefix+text) + "\n"
}

// FormatSuccessMessage formats a success message with optional emoji
func FormatSuccessMessage(message, emoji string) string {
	prefix := ""
	if emoji != "" {
		prefix = emoji + " "
	}
	result := SuccessStyle.Render(prefix + message)
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

// FormatUsage formats usage information in gray
func FormatUsage(message string) string {
	return GrayStyle.Render(message) + "\n"
}

// FormatListBullet formats a bullet list item
func FormatListBullet(text string) string {
	return ListStyle.Render("  - ") + text + "\n"
}

// FormatListBulletWithStyle formats a bullet list item with custom style
func FormatListBulletWithStyle(text string, style lipgloss.Style) string {
	return ListStyle.Render("  - ") + style.Render(text) + "\n"
}

// isPrivateIP checks if the given IP address is in a private network range
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	
	// Check private IPv4 ranges:
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}
	
	for _, block := range privateBlocks {
		_, ipNet, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	
	return false
}

// FormatIP formats an IP address for display with color
// Local IPs (private network) are styled in yellow/orange, internet IPs in cyan
func FormatIP(ip string) string {
	var color string
	if isPrivateIP(ip) {
		color = currentTheme.Warning // Yellow/Orange (220) for local IPs
	} else {
		color = currentTheme.Info // Cyan (51) for internet IPs
	}
	
	ipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return ipStyle.Render(ip)
}

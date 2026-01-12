package terminal

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"

	"terminal-sh/database"
	"terminal-sh/filesystem"
	"terminal-sh/models"

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

// Emoji constants organized by category
const (
	// Filesystem
	EmojiFile      = "📄"
	EmojiFolder    = "📁"
	EmojiEdit      = "✏️"
	
	// User
	EmojiUser      = "👤"
	EmojiProfile   = " profile"
	
	// Network
	EmojiServer    = "🖥️"
	EmojiNetwork   = "🌐"
	EmojiIP        = "📍"
	EmojiSSH       = "🔌"
	EmojiScan      = "🔍"
	
	// Tools
	EmojiTool      = "🛠️"
	EmojiTools     = "🛠️"
	EmojiExploit   = "⚡"
	EmojiHack      = "💻"
	
	// Shop
	EmojiShop      = "🛒"
	EmojiMoney     = "💰"
	EmojiBuy       = "🛍️"
	
	// System
	EmojiInfo      = "ℹ️"
	EmojiSuccess   = "✅"
	EmojiError     = "❌"
	EmojiWarning   = "⚠️"
	EmojiHelp      = "❓"
	
	// Learning
	EmojiTutorial  = "📚"
	EmojiLearning  = "🎓"
	
	// Upgrades
	EmojiPatch     = "🔧"
	EmojiUpgrade   = "⬆️"
	
	// Mining
	EmojiMining    = "⛏️"
	EmojiCrypto    = "₿"
)

var (
	// Styles using current theme
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(currentTheme.Primary)).
			Bold(true)

	dirStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Secondary))

	fileStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.LightGray))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(currentTheme.Error)).
			Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Success))
	
	// New style variables
	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Primary)).
		Bold(true)
	
	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Info))
	
	valueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.LightGray))
	
	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Info))
	
	warningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Warning)).
		Bold(true)
	
	listItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Secondary))
)

// RenderPrompt renders the shell prompt with styled user, hostname, and path.
func RenderPrompt(user, hostname, path string) string {
	prompt := fmt.Sprintf("%s@%s:%s$ ", user, hostname, path)
	return promptStyle.Render(prompt)
}

// getFileTypeInfo returns emoji and color for a file based on its extension
func getFileTypeInfo(filename string) (emoji string, color string) {
	ext := strings.ToLower(filepath.Ext(filename))
	
	// Directories
	if ext == "" && strings.HasSuffix(filename, "/") {
		return EmojiFolder, currentTheme.Secondary
	}
	
	// Different file types with colors and emojis
	switch ext {
	// Code files
	case ".go":
		return "🔷", "81" // Cyan blue
	case ".js", ".jsx":
		return "🟨", "220" // Yellow
	case ".ts", ".tsx":
		return "🔵", "75" // Light blue
	case ".py":
		return "🐍", "208" // Orange
	case ".java":
		return "☕", "208" // Orange
	case ".cpp", ".cxx", ".cc":
		return "⚙️", "39" // Blue
	case ".c":
		return "⚙️", "39" // Blue
	case ".rs":
		return "🦀", "196" // Red
	case ".php":
		return "🐘", "105" // Purple
	case ".rb":
		return "💎", "196" // Red
	case ".sh", ".bash", ".zsh":
		return "💻", "46" // Green
	case ".ps1":
		return "🔷", "51" // Cyan
	// Markup/Config
	case ".html", ".htm":
		return "🌐", "202" // Orange red
	case ".css":
		return "🎨", "51" // Cyan
	case ".json":
		return "📋", "220" // Yellow
	case ".yaml", ".yml":
		return "📝", "51" // Cyan
	case ".xml":
		return "📄", "220" // Yellow
	case ".toml":
		return "⚙️", "252" // Light gray
	case ".ini", ".conf", ".config":
		return "⚙️", "240" // Gray
	// Text files
	case ".txt", ".text":
		return "📄", "252" // Light gray
	case ".md", ".markdown":
		return "📖", "252" // Light gray
	case ".readme":
		return "📚", "252" // Light gray
	case ".log":
		return "📋", "240" // Gray
	// Archives
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar":
		return "📦", "208" // Orange
	// Images
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".bmp", ".ico":
		return "🖼️", "51" // Cyan
	// Audio
	case ".mp3", ".wav", ".ogg", ".flac", ".aac":
		return "🎵", "213" // Pink
	// Video
	case ".mp4", ".avi", ".mov", ".mkv", ".webm":
		return "🎬", "213" // Pink
	// Documents
	case ".pdf":
		return "📕", "196" // Red
	case ".doc", ".docx":
		return "📘", "51" // Cyan
	case ".xls", ".xlsx":
		return "📗", "46" // Green
	case ".ppt", ".pptx":
		return "📙", "208" // Orange
	// Executables (no extension or .exe)
	case ".exe", ".bin", ".app":
		return "⚡", "196" // Red
	default:
		// No extension or unknown extension
		if ext == "" {
			return EmojiFile, currentTheme.LightGray
		}
		return EmojiFile, currentTheme.LightGray
	}
}

// FormatDirList formats a list of filesystem nodes for display.
// Returns output with trailing newline, or empty string if no nodes.
func FormatDirList(nodes []*filesystem.Node) string {
	return FormatDirListWithOptions(nodes, false)
}

// FormatDirListWithOptions formats a list with optional long format (detailed view).
func FormatDirListWithOptions(nodes []*filesystem.Node, longFormat bool) string {
	if len(nodes) == 0 {
		return ""
	}

	var output strings.Builder
	for _, node := range nodes {
		if longFormat {
			// Long format: show file details with colors and emojis
			// Format: permissions size emoji name
			perms := "-rw-r--r--"
			if node.IsDir {
				perms = "drwxr-xr-x"
			}
			size := "0"
			if !node.IsDir {
				size = fmt.Sprintf("%d", len(node.Content))
			}
			
			name := node.Name
			emoji, color := getFileTypeInfo(name)
			if node.IsDir {
				emoji = EmojiFolder
				color = currentTheme.Secondary
				name += "/"
			}
			
			nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
			output.WriteString(fmt.Sprintf("%s %6s %s %s\n", perms, size, emoji, nameStyle.Render(name)))
		} else {
			// Short format: just colored name (no emoji)
			if node.IsDir {
				output.WriteString(dirStyle.Render(node.Name + "/"))
			} else {
				_, color := getFileTypeInfo(node.Name)
				fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
				output.WriteString(fileStyle.Render(node.Name))
			}
			output.WriteString("\n")
		}
	}

	// Ensure trailing newline
	result := output.String()
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

// FormatError formats an error message for display
// Returns error message with trailing newline
func FormatError(err error) string {
	msg := errorStyle.Render("Error: " + err.Error())
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return msg
}

// FormatSuccess formats a success message for display
// Returns success message with trailing newline (or empty string if msg is empty)
func FormatSuccess(msg string) string {
	if msg == "" {
		return ""
	}
	result := successStyle.Render(msg)
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
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

// FormatHeader formats a section header with optional emoji
func FormatHeader(title, emoji string) string {
	if emoji != "" {
		title = emoji + " " + title
	}
	return headerStyle.Render(title) + "\n"
}

// FormatLabel formats a field label
func FormatLabel(label string) string {
	return labelStyle.Render(label)
}

// FormatValue formats a data value
func FormatValue(value string) string {
	return valueStyle.Render(value)
}

// FormatKeyValue formats a key-value pair with optional color
func FormatKeyValue(key, value, color string) string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(currentTheme.Info))
	if color != "" {
		valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		return keyStyle.Render(key) + ": " + valStyle.Render(value)
	}
	return keyStyle.Render(key) + ": " + valueStyle.Render(value)
}

// FormatListItem formats a list item with optional emoji
func FormatListItem(text, emoji string) string {
	prefix := "  "
	if emoji != "" {
		prefix = fmt.Sprintf("  %s ", emoji)
	}
	return listItemStyle.Render(prefix+text) + "\n"
}

// FormatSuccessWithEmoji formats a success message with optional emoji
func FormatSuccessWithEmoji(message, emoji string) string {
	if message == "" {
		return ""
	}
	prefix := ""
	if emoji != "" {
		prefix = emoji + " "
	}
	result := successStyle.Render(prefix + message)
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

// FormatInfo formats an informational message
func FormatInfo(message string) string {
	result := infoStyle.Render(message)
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

// FormatBox formats boxed content with a title
func FormatBox(title, content string) string {
	width := 45
	if len(title) > width-4 {
		width = len(title) + 4
	}
	
	var sb strings.Builder
	sb.WriteString("╔" + strings.Repeat("═", width-2) + "╗\n")
	titlePadded := title
	if len(titlePadded) < width-4 {
		titlePadded = "   " + titlePadded + strings.Repeat(" ", width-4-len(titlePadded))
	}
	sb.WriteString("║" + titlePadded + "║\n")
	sb.WriteString("╚" + strings.Repeat("═", width-2) + "╝\n\n")
	sb.WriteString(content)
	
	return sb.String()
}

// AnimatedWelcome returns an animated "TERMINAL.SH" ASCII art welcome message
func AnimatedWelcome() string {
	// ASCII art for TERMINAL.SH (proper block letters)
	asciiArt := `
████████╗███████╗██████╗ ███╗   ███╗██╗███╗   ██╗ █████╗ ██╗         ███████╗██╗  ██╗
╚══██╔══╝██╔════╝██╔══██╗████╗ ████║██║████╗  ██║██╔══██╗██║         ██╔════╝██║  ██║
   ██║   █████╗  ██████╔╝██╔████╔██║██║██╔██╗ ██║███████║██║         ███████╗███████║
   ██║   ██╔══╝  ██╔══██╗██║╚██╔╝██║██║██║╚██╗██║██╔══██║██║         ╚════██║██╔══██║
   ██║   ███████╗██║  ██║██║ ╚═╝ ██║██║██║ ╚████║██║  ██║███████╗    ███████║██║  ██║
   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝    ╚══════╝╚═╝  ╚═╝
`
	
	var styled strings.Builder
	
	// Style the ASCII art with gradient colors
	lines := strings.Split(strings.TrimPrefix(asciiArt, "\n"), "\n")
	colors := []string{"205", "213", "207", "219", "218", "212", "205"} // Magenta/pink gradient
	
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

// WelcomeHelpText returns the help text that appears below the ASCII art
// If user is provided, shows conditional message (first login vs last login)
func WelcomeHelpText(user *models.User, db *database.Database) string {
	var styled strings.Builder
	
	styled.WriteString("\n")
	
	// Subtitle with instructions - use plain ANSI codes for consistent alignment
	styled.WriteString("\x1b[38;5;39m\x1b[3mType 'help' for available commands\x1b[0m\n")
	
	// Conditional note based on first login or last login
	if user != nil && db != nil {
		// Check if this is a first login (user created within last 2 minutes)
		isFirstLogin := time.Since(user.CreatedAt) < 2*time.Minute
		
		if isFirstLogin {
			// First login - show auto-created message
			styled.WriteString("\x1b[38;5;240m\x1b[2mNote: Your account was auto-created on first login\x1b[0m")
		} else {
			// Not first login - show last login info
			// Query for the most recent session before this one (exclude sessions from last 30 seconds to avoid current session)
			var lastSession models.Session
			cutoffTime := time.Now().Add(-30 * time.Second)
			if err := db.Where("user_id = ? AND created_at < ?", user.ID, cutoffTime).Order("created_at DESC").First(&lastSession).Error; err == nil {
				// Found a previous session - show last login time
				lastLoginTime := lastSession.CreatedAt.Format("Mon Jan 2 15:04:05 MST 2006")
				styled.WriteString("\x1b[38;5;39mLast login: \x1b[38;5;252m" + lastLoginTime + "\x1b[0m")
			} else {
				// No previous session found, but account is old - show account creation date
				createdTime := user.CreatedAt.Format("Mon Jan 2 15:04:05 MST 2006")
				styled.WriteString("\x1b[38;5;39mAccount created: \x1b[38;5;252m" + createdTime + "\x1b[0m")
			}
		}
	} else {
		// Fallback if no user/db provided
		styled.WriteString("\x1b[38;5;240m\x1b[2mNote: Your account was auto-created on first login\x1b[0m")
	}
	
	return styled.String()
}


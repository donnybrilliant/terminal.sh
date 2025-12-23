package terminal

import (
	"fmt"
	"strings"

	"ssh4xx-go/filesystem"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	dirStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	fileStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("46"))
)

// RenderPrompt renders the shell prompt
func RenderPrompt(user, hostname, path string) string {
	prompt := fmt.Sprintf("%s@%s:%s$ ", user, hostname, path)
	return promptStyle.Render(prompt)
}

// FormatDirList formats a list of filesystem nodes for display
func FormatDirList(nodes []*filesystem.Node) string {
	if len(nodes) == 0 {
		return ""
	}

	var output strings.Builder
	for _, node := range nodes {
		if node.IsDir {
			output.WriteString(dirStyle.Render(node.Name + "/"))
		} else {
			output.WriteString(fileStyle.Render(node.Name))
		}
		output.WriteString("\n")
	}

	return strings.TrimSuffix(output.String(), "\n")
}

// FormatError formats an error message for display
func FormatError(err error) string {
	return errorStyle.Render("Error: " + err.Error())
}

// FormatSuccess formats a success message for display
func FormatSuccess(msg string) string {
	return successStyle.Render(msg)
}

// AnimatedWelcome returns an animated "SSH4XX" ASCII art welcome message
func AnimatedWelcome() string {
	// ASCII art for SSH4XX (proper block letters)
	asciiArt := `
███████╗███████╗██╗  ██╗██╗  ██╗██╗  ██╗██╗  ██╗
██╔════╝██╔════╝██║  ██║██║  ██║╚██╗██╔╝╚██╗██╔╝
███████╗███████╗███████║███████║ ╚███╔╝  ╚███╔╝ 
╚════██║╚════██║██╔══██║╚════██║ ██╔██╗  ██╔██╗ 
███████║███████║██║  ██║     ██║██╔╝ ██╗██╔╝ ██╗
╚══════╝╚══════╝╚═╝  ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝
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
func WelcomeHelpText() string {
	var styled strings.Builder
	
	styled.WriteString("\n")
	
	// Subtitle with instructions
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Italic(true)
	styled.WriteString(subtitleStyle.Render("Type 'help' for available commands\n"))
	
	noteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Faint(true)
	styled.WriteString(noteStyle.Render("Note: Your account was auto-created on first login"))
	
	return styled.String()
}

// AnimatedHelp returns help text with color transitions
func AnimatedHelp() string {
	// Define sections with their commands
	sections := []struct {
		title  string
		titleColor string
		commands []struct {
			cmd     string
			desc    string
			color   string
		}
	}{
		{
			title: "Filesystem",
			titleColor: "39", // Blue
			commands: []struct{cmd, desc, color string}{
				{"pwd", "Print working directory", "46"},
				{"ls", "List directory contents", "51"},
				{"cd", "Change directory", "45"},
				{"cat", "Display file contents", "213"},
				{"touch", "Create a new file", "207"},
				{"mkdir", "Create a new directory", "39"},
				{"rm", "Delete file (rm -r for directory)", "46"},
				{"cp", "Copy files/folders", "51"},
				{"mv", "Move or rename files/folders", "45"},
				{"edit", "Edit a file", "213"},
			},
		},
		{
			title: "User",
			titleColor: "46", // Green
			commands: []struct{cmd, desc, color string}{
				{"userinfo", "Show user information", "51"},
				{"whoami", "Display current username", "45"},
				{"name", "Change username", "213"},
			},
		},
		{
			title: "Network",
			titleColor: "51", // Cyan
			commands: []struct{cmd, desc, color string}{
				{"ifconfig", "Show network interfaces", "45"},
				{"scan", "Scan internet or IP", "213"},
				{"ssh", "Connect to a server", "207"},
				{"exit", "Disconnect from server", "39"},
				{"server", "Show current server info", "46"},
				{"createServer", "Create a new server", "51"},
				{"createLocalServer", "Create local server", "45"},
			},
		},
		{
			title: "Tools",
			titleColor: "213", // Magenta
			commands: []struct{cmd, desc, color string}{
				{"get", "Download tool from server", "207"},
				{"tools", "List owned tools", "39"},
				{"exploited", "List exploited servers", "46"},
				{"wallet", "Show wallet balance", "51"},
			},
		},
		{
			title: "Tool Commands",
			titleColor: "207", // Pink
			commands: []struct{cmd, desc, color string}{
				{"password_cracker", "Crack passwords", "39"},
				{"ssh_exploit", "Exploit SSH vulnerabilities", "46"},
				{"user_enum", "Enumerate users", "51"},
				{"lan_sniffer", "Discover network connections", "45"},
				{"rootkit", "Install backdoor", "213"},
				{"exploit_kit", "Multi-vulnerability exploit", "207"},
				{"crypto_miner", "Start mining", "39"},
				{"stop_mining", "Stop mining", "46"},
				{"miners", "List active miners", "51"},
			},
		},
		{
			title: "System",
			titleColor: "252", // Light gray
			commands: []struct{cmd, desc, color string}{
				{"clear", "Clear the screen", "39"},
				{"help", "Show this help message", "46"},
			},
		},
	}

	var styled strings.Builder
	
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginBottom(1)
	styled.WriteString(titleStyle.Render("Available Commands") + "\n\n")

	// Render each section
	for _, section := range sections {
		sectionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(section.titleColor)).
			Bold(true)
		styled.WriteString(sectionStyle.Render(section.title + ":") + "\n")
		
		for _, cmd := range section.commands {
			cmdStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(cmd.color))
			// Format: "  command - description"
			// Pad command to 20 chars for alignment
			cmdPadded := cmd.cmd
			if len(cmdPadded) < 20 {
				cmdPadded += strings.Repeat(" ", 20-len(cmdPadded))
			}
			styled.WriteString("  " + cmdStyle.Render(cmdPadded) + " - " + cmd.desc + "\n")
		}
		styled.WriteString("\n")
	}

	return styled.String()
}


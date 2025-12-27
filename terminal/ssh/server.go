package ssh

import (
	"context"
	"fmt"
	"terminal-sh/config"
	"terminal-sh/database"
	"terminal-sh/services"
	"terminal-sh/terminal"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	wishbubbletea "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

var (
	successLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
	infoLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)

var (
	sshServer *ssh.Server
	serverMu   sync.Mutex
)

// StartServer starts the SSH server using Wish framework
// No SSH authentication - everyone connects and sees the Bubble Tea login form
func StartServer(cfg *config.Config, db *database.Database) error {
	userService := services.NewUserService(db, cfg.JWTSecret)

	// Use default host key path if not provided
	hostKeyPath := cfg.HostKeyPath
	if hostKeyPath == "" {
		hostKeyPath = ".ssh/ssh_host_key"
	}

	// Create Wish server with NO SSH authentication
	// SSH is just secure transport - app handles authentication
	// By not providing any auth callbacks, all connections are allowed
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithMiddleware(
			// Logging middleware
			logging.Middleware(),
			// Bubble Tea middleware - shows login form to everyone
			wishbubbletea.Middleware(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				// Extract username from SSH session (if provided)
				// SSH protocol requires a username, but we ignore it for auth
				// We'll use it as a hint/prefill in the login form
				username := sess.User()
				
				// If username is empty or "guest", use empty string for prefill
				if username == "" || username == "guest" {
					username = ""
				}
				
				// Create login model with prefilled username (no password from SSH)
				// Everyone sees the login form - no SSH auth required
				model := terminal.NewLoginModel(db, userService, username, "")
				
				// After login, transition to shell
				return model, []tea.ProgramOption{tea.WithAltScreen()}
			}),
		),
		// No authentication callbacks = no SSH auth required
		// All connections are allowed and go directly to login form
	)

	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Store server reference for graceful shutdown
	serverMu.Lock()
	sshServer = s
	serverMu.Unlock()

	fmt.Println(infoLogStyle.Render(fmt.Sprintf("SSH server listening on %s:%d", cfg.Host, cfg.Port)))
	fmt.Println(successLogStyle.Render("✓") + " No SSH authentication required")
	fmt.Println(successLogStyle.Render("✓") + " All connections go directly to login form")
	fmt.Println(successLogStyle.Render("✓") + " " + infoLogStyle.Render(fmt.Sprintf("Users can connect with: ssh user@host -p %d", cfg.Port)))
	fmt.Println(successLogStyle.Render("✓") + " Press Ctrl+C to shutdown gracefully")
	
	return s.ListenAndServe()
}

// ShutdownServer gracefully shuts down the SSH server
func ShutdownServer(ctx context.Context) error {
	serverMu.Lock()
	s := sshServer
	serverMu.Unlock()

	if s == nil {
		return nil // Server not started
	}

	fmt.Println(infoLogStyle.Render("Shutting down SSH server..."))
	return s.Shutdown(ctx)
}


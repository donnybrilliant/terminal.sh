package terminal

import (
	"fmt"
	"strings"
	"terminal-sh/models"
	"terminal-sh/services"

	"terminal-sh/database"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	welcomeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)

// LoginModel handles the login/registration form
type LoginModel struct {
	db            *database.Database
	form          *huh.Form
	userService   *services.UserService
	chatService   *services.ChatService
	username      string
	password      string
	prefillUser   string // Username from SSH connection
	prefillPass   string // Password from SSH connection (if provided)
	authenticated bool
	user          interface{} // Will store the authenticated user
	err           error
	width         int
	height        int
	formWidth     int // Store form width separately
}

// NewLoginModel creates a new login model
func NewLoginModel(db *database.Database, userService *services.UserService, chatService *services.ChatService, prefillUsername, prefillPassword string) *LoginModel {
	model := &LoginModel{
		db:          db,
		userService: userService,
		chatService: chatService,
		username:    prefillUsername,
		password:    prefillPassword,
		prefillUser: prefillUsername,
		prefillPass: prefillPassword,
		width:       80,  // Default
		height:      24,  // Default
		formWidth:   50,  // Default form width
	}

	// Form will be resized based on window size
	// Use model fields directly so form updates them
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Value(&model.username).
				Placeholder("Enter your username").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("username cannot be empty")
					}
					return nil
				}),
			huh.NewInput().
				Title("Password").
				Value(&model.password).
				Placeholder("Enter your password").
				EchoMode(huh.EchoModePassword).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("password cannot be empty")
					}
					return nil
				}),
		),
	).
		WithTheme(huh.ThemeCatppuccin()).
		WithWidth(50) // Default width, will be updated on window resize

	model.form = form
	return model
}

// Init initializes the model
func (m *LoginModel) Init() tea.Cmd {
	// If we have both username and password prefilled, try auto-login
	if m.prefillUser != "" && m.prefillPass != "" {
		return m.attemptAutoLogin()
	}
	return m.form.Init()
}

// attemptAutoLogin tries to login automatically if credentials are provided
func (m *LoginModel) attemptAutoLogin() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			user, _, err := m.userService.Login(m.prefillUser, m.prefillPass)
			if err != nil {
				// User doesn't exist, try to register
				user, err = m.userService.Register(m.prefillUser, m.prefillPass)
				if err != nil {
					return LoginErrorMsg{Error: err}
				}
			}
			return LoginSuccessMsg{User: user}
		},
	)
}

// Update handles messages
func (m *LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update window dimensions for responsive centering
		m.width = msg.Width
		m.height = msg.Height
		// Update form width (use 60% of window width, min 40, max 80)
		formWidth := int(float64(msg.Width) * 0.6)
		if formWidth < 40 {
			formWidth = 40
		}
		if formWidth > 80 {
			formWidth = 80
		}
		m.formWidth = formWidth
		m.form = m.form.WithWidth(formWidth)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			// Allow quitting from login
			return m, tea.Quit
		}

	case LoginSuccessMsg:
		m.authenticated = true
		m.user = msg.User
		m.err = nil // Clear any previous errors
		// Store user in context for shell to use
		// Transition to shell model with current window size
		shellModel := NewShellModelWithSize(m.db, m.userService, msg.User, m.width, m.height, m.chatService)
		return shellModel, shellModel.Init()

	case LoginErrorMsg:
		// Show generic error message for security
		m.err = fmt.Errorf("invalid username or password")
		// Reset form state to allow retry
		m.form.State = huh.StateNormal
		// Clear password field for security
		m.password = ""
		// Recreate form to reset it - use model fields directly
		m.form = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Username").
					Value(&m.username).
					Placeholder("Enter your username").
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("username cannot be empty")
						}
						return nil
					}),
				huh.NewInput().
					Title("Password").
					Value(&m.password).
					Placeholder("Enter your password").
					EchoMode(huh.EchoModePassword).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("password cannot be empty")
						}
						return nil
					}),
			),
		).
			WithTheme(huh.ThemeCatppuccin()).
			WithWidth(m.formWidth)
		return m, m.form.Init()
	}

	if !m.authenticated {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
			// Update username/password from form
			// Note: Huh forms update the pointers directly
		}

		// Check if form is complete
		if m.form.State == huh.StateCompleted {
			// Clear any previous errors
			m.err = nil
			return m, m.handleSubmit()
		}

		return m, cmd
	}

	return m, nil
}

// handleSubmit processes the form submission
func (m *LoginModel) handleSubmit() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			// Try to login first
			user, _, err := m.userService.Login(m.username, m.password)
			if err != nil {
			// Check if it's a password error (user exists but wrong password)
			// vs user doesn't exist (auto-register)
			var existingUser models.User
			err := m.db.Where("username = ?", m.username).First(&existingUser).Error
			userExists := err == nil
				
				if userExists {
					// User exists but password is wrong - return generic error
					return LoginErrorMsg{Error: fmt.Errorf("invalid username or password")}
				}
				
				// User doesn't exist, try to register (auto-create)
				user, err = m.userService.Register(m.username, m.password)
				if err != nil {
					// Registration failed - return generic error for security
					return LoginErrorMsg{Error: fmt.Errorf("invalid username or password")}
				}
			}
			return LoginSuccessMsg{User: user}
		},
	)
}

// View renders the UI
func (m *LoginModel) View() string {
	if m.authenticated {
		return ""
	}

	// Ensure minimum dimensions
	width := m.width
	height := m.height
	if width < 40 {
		width = 80
	}
	if height < 10 {
		height = 24
	}

	// Build content
	var content strings.Builder
	
	// Title box
	title := `╔═══════════════════════════════════════╗
║   terminal.sh Server - Welcome!       ║
╚═══════════════════════════════════════╝`
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")
	content.WriteString(welcomeStyle.Render("Enter your credentials to continue"))
	content.WriteString("\n\n")

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		content.WriteString("\n\n")
	}

	content.WriteString(m.form.View())

	// Use lipgloss.Place() to center content in full terminal frame
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		content.String(),
	)
}

// Messages
type LoginSuccessMsg struct {
	User interface{}
}

type LoginErrorMsg struct {
	Error error
}

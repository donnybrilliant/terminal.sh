package terminal

import (
	"fmt"
	"terminal-sh/models"
	"terminal-sh/services"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"terminal-sh/database"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Align(lipgloss.Center).
			MarginBottom(1)

	welcomeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Align(lipgloss.Center).
			MarginBottom(2)
)

// LoginModel handles the login/registration form
type LoginModel struct {
	form          *huh.Form
	userService   *services.UserService
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
func NewLoginModel(userService *services.UserService, prefillUsername, prefillPassword string) *LoginModel {
	username := prefillUsername
	password := prefillPassword

	// Form will be resized based on window size
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Value(&username).
				Placeholder("Enter your username").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("username cannot be empty")
					}
					return nil
				}),
			huh.NewInput().
				Title("Password").
				Value(&password).
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

	return &LoginModel{
		form:        form,
		userService: userService,
		username:    username,
		password:    password,
		prefillUser: prefillUsername,
		prefillPass: prefillPassword,
		width:       80,  // Default
		height:      24,  // Default
		formWidth:   50,  // Default form width
	}
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
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case LoginSuccessMsg:
		m.authenticated = true
		m.user = msg.User
		m.err = nil // Clear any previous errors
		// Store user in context for shell to use
		// Transition to shell model with current window size
		shellModel := NewShellModelWithSize(m.userService, msg.User, m.width, m.height)
		return shellModel, shellModel.Init()

	case LoginErrorMsg:
		// Show generic error message for security
		m.err = fmt.Errorf("invalid username or password")
		// Reset form state to allow retry
		m.form.State = huh.StateNormal
		// Clear password field for security
		m.password = ""
		// Recreate form to reset it
		username := m.username
		password := ""
		m.form = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Username").
					Value(&username).
					Placeholder("Enter your username").
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("username cannot be empty")
						}
						return nil
					}),
				huh.NewInput().
					Title("Password").
					Value(&password).
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
		m.username = username
		m.password = password
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
			err := database.DB.Where("username = ?", m.username).First(&existingUser).Error
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

	var s string
	s += titleStyle.Render("╔═══════════════════════════════════════╗\n║   terminal.sh Server - Welcome!       ║\n╚═══════════════════════════════════════╝")
	s += "\n\n"
	s += welcomeStyle.Render("Enter your credentials to continue")
	s += "\n\n"

	if m.err != nil {
		s += lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error: %v\n\n", m.err))
	}

	s += m.form.View()
	s += "\n\n"
	s += lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("Press Ctrl+C or 'q' to quit")

	// Center the content based on actual window size
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		s,
	)
}

// Messages
type LoginSuccessMsg struct {
	User interface{}
}

type LoginErrorMsg struct {
	Error error
}


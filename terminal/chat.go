package terminal

import (
	"fmt"
	"strings"
	"time"

	"terminal-sh/models"
	"terminal-sh/services"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

// Use time package to avoid unused import error
var _ = time.Time{}

// ChatModel handles the chat UI
type ChatModel struct {
	parent        *ShellModel
	chatService   *services.ChatService
	user          *models.User
	activeRoomID  uuid.UUID
	rooms         []*models.ChatRoom
	roomViewports map[uuid.UUID]*viewport.Model
	roomMessages  map[uuid.UUID][]models.ChatMessage
	tabIndex      int
	textInput     textinput.Model
	width         int
	height        int
	msgChan       chan models.ChatMessage
	sessionID     uuid.UUID
	splitMode     bool
	inputHistory  *InputHistory // Command history for up/down navigation
}

// ChatMessageMsg is sent when a new message arrives
type ChatMessageMsg struct {
	Message models.ChatMessage
}

// chatExitMsg signals exiting chat back to shell
type chatExitMsg struct{}

// NewChatModel creates a new chat model
func NewChatModel(parent *ShellModel, chatService *services.ChatService, user *models.User, sessionID uuid.UUID, width, height int, splitMode bool) *ChatModel {
	// Register session with chat service
	msgChan := chatService.RegisterSession(sessionID, user.ID)

	// Get user's rooms
	rooms, _ := chatService.GetRoomsForUser(user.ID)

	// Auto-join #public if not in any rooms
	if len(rooms) == 0 {
		publicRoom, err := chatService.GetRoomByName("#public")
		if err == nil {
			chatService.JoinRoom(publicRoom.ID, user.ID, "")
			rooms, _ = chatService.GetRoomsForUser(user.ID)
		}
	}

	activeRoomID := uuid.Nil
	if len(rooms) > 0 {
		activeRoomID = rooms[0].ID
	}

	// Initialize text input
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Type a message or /help for commands..."
	ti.CharLimit = 0
	ti.Width = width
	ti.Focus()

	model := &ChatModel{
		parent:        parent,
		chatService:   chatService,
		user:          user,
		activeRoomID:  activeRoomID,
		rooms:         rooms,
		roomViewports: make(map[uuid.UUID]*viewport.Model),
		roomMessages:  make(map[uuid.UUID][]models.ChatMessage),
		tabIndex:      0,
		textInput:     ti,
		width:         width,
		height:        height,
		msgChan:       msgChan,
		sessionID:     sessionID,
		splitMode:     splitMode,
		inputHistory:  NewInputHistory(100), // Keep last 100 commands
	}

	// Initialize viewports and load messages for each room
	for _, room := range rooms {
		vp := viewport.New(width, height-5) // Reserve space for top padding (1) + tabs (2) + input (1) + buffer (1)
		model.roomViewports[room.ID] = &vp

		// Load recent messages
		messages, _ := chatService.GetRecentMessages(room.ID, 100)
		model.roomMessages[room.ID] = messages
		model.updateViewportContent(room.ID)
	}

	return model
}

// Init initializes the chat model
func (m *ChatModel) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		m.textInput.Focus(),
		m.listenForMessages(),
	)
}

// listenForMessages listens to the message channel
func (m *ChatModel) listenForMessages() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.msgChan
		if !ok {
			return nil
		}
		return ChatMessageMsg{Message: msg}
	}
}

// Update handles messages
func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width

		// Update all viewports
		for roomID := range m.roomViewports {
			vp := m.roomViewports[roomID]
			vp.Width = msg.Width
			vp.Height = msg.Height - 5 // Reserve space for top padding (1) + tabs (2) + input (1) + buffer (1)
			m.updateViewportContent(roomID)
		}

		return m, nil

	case ChatMessageMsg:
		// New message arrived
		msgRoomID := msg.Message.RoomID

		// Add to messages
		if m.roomMessages[msgRoomID] == nil {
			m.roomMessages[msgRoomID] = make([]models.ChatMessage, 0)
		}
		m.roomMessages[msgRoomID] = append(m.roomMessages[msgRoomID], msg.Message)

		// Trim if too many
		if len(m.roomMessages[msgRoomID]) > 100 {
			m.roomMessages[msgRoomID] = m.roomMessages[msgRoomID][len(m.roomMessages[msgRoomID])-100:]
		}

		// Update viewport if this is the active room
		if msgRoomID == m.activeRoomID {
			m.updateViewportContent(msgRoomID)
		}

		// Continue listening
		cmds = append(cmds, m.listenForMessages())

		return m, tea.Batch(cmds...)

	case chatExitMsg:
		return m.parent, nil

	case tea.KeyMsg:
		// Handle special keys
		switch msg.String() {
		case "ctrl+c", "esc":
			// Exit chat mode
			m.chatService.UnregisterSession(m.sessionID)
			return m.parent, nil

		case "left":
			// Switch to previous tab
			if len(m.rooms) > 0 {
				m.tabIndex--
				if m.tabIndex < 0 {
					m.tabIndex = len(m.rooms) - 1
				}
				m.activeRoomID = m.rooms[m.tabIndex].ID
				m.updateViewportContent(m.activeRoomID)
			}
			return m, nil

		case "right":
			// Switch to next tab
			if len(m.rooms) > 0 {
				m.tabIndex++
				if m.tabIndex >= len(m.rooms) {
					m.tabIndex = 0
				}
				m.activeRoomID = m.rooms[m.tabIndex].ID
				m.updateViewportContent(m.activeRoomID)
			}
			return m, nil

		case "tab":
			// Check if we should autocomplete a command
			input := m.textInput.Value()
			if strings.HasPrefix(input, "/") {
				// Autocomplete command
				completed := m.autocompleteCommand(input)
				if completed != input {
					m.textInput.SetValue(completed)
					m.textInput.CursorEnd()
					return m, nil
				}
			}
			// Otherwise cycle through tabs
			if len(m.rooms) > 0 {
				m.tabIndex = (m.tabIndex + 1) % len(m.rooms)
				m.activeRoomID = m.rooms[m.tabIndex].ID
				m.updateViewportContent(m.activeRoomID)
			}
			return m, nil

		case "up":
			// Command history - previous
			if cmd, ok := m.inputHistory.Previous(); ok {
				m.textInput.SetValue(cmd)
				m.textInput.CursorEnd()
			}
			return m, nil

		case "down":
			// Command history - next
			if cmd, ok := m.inputHistory.Next(); ok {
				m.textInput.SetValue(cmd)
				m.textInput.CursorEnd()
			} else {
				m.textInput.SetValue("")
			}
			return m, nil

		case "pgup", "pgdown":
			// Scroll viewport with page up/down
			if vp, ok := m.roomViewports[m.activeRoomID]; ok {
				var cmd tea.Cmd
				*vp, cmd = vp.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case "enter":
			// Send message or execute command
			input := m.textInput.Value()
			m.textInput.SetValue("")
			m.inputHistory.Reset()

			if input == "" {
				return m, nil
			}

			// Check if it's a command - only commands go in history
			if strings.HasPrefix(input, "/") {
				m.inputHistory.Add(input)
				return m, m.handleCommand(input)
			}

			// Send message to active room (not added to history)
			if m.activeRoomID != uuid.Nil {
				err := m.chatService.SendMessage(m.activeRoomID, m.user.ID, m.user.Username, input)
				if err != nil {
					// Error will be shown in next update
					return m, nil
				}
			}

			return m, nil

		default:
			// Handle text input
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleCommand processes chat commands
func (m *ChatModel) handleCommand(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/join":
		if len(args) < 1 {
			m.addSystemMessage("Usage: /join <room> [password]")
			return nil
		}
		roomName := args[0]
		password := ""
		if len(args) > 1 {
			password = args[1]
		}

		// Check if already in this room
		for _, r := range m.rooms {
			if r.Name == roomName {
				m.addSystemMessage("Already in room " + roomName)
				return nil
			}
		}

		// Get existing room (use /create for new rooms)
		room, err := m.chatService.GetRoomByName(roomName)
		if err != nil {
			m.addSystemMessage("Room not found: " + roomName + " (use /create to make a new room)")
			return nil
		}

		// Join room
		err = m.chatService.JoinRoom(room.ID, m.user.ID, password)
		if err != nil {
			m.addSystemMessage("Cannot join: " + err.Error())
			return nil
		}

		// Add to rooms list
		m.rooms = append(m.rooms, room)
		m.roomMessages[room.ID] = make([]models.ChatMessage, 0)
		vp := viewport.New(m.width, m.height-5)
		m.roomViewports[room.ID] = &vp

		// Load messages
		messages, _ := m.chatService.GetRecentMessages(room.ID, 100)
		m.roomMessages[room.ID] = messages
		m.updateViewportContent(room.ID)

		// Switch to new room
		m.tabIndex = len(m.rooms) - 1
		m.activeRoomID = room.ID
		m.addSystemMessage("Joined " + roomName)

	case "/leave":
		// Use current room if no args
		var roomName string
		if len(args) < 1 {
			if m.tabIndex >= 0 && m.tabIndex < len(m.rooms) {
				roomName = m.rooms[m.tabIndex].Name
			} else {
				m.addSystemMessage("No room to leave")
				return nil
			}
		} else {
			roomName = args[0]
		}

		// Find room
		var roomIndex int = -1
		var roomID uuid.UUID
		for i, room := range m.rooms {
			if room.Name == roomName {
				roomIndex = i
				roomID = room.ID
				break
			}
		}

		if roomIndex < 0 {
			m.addSystemMessage("Not in room " + roomName)
			return nil
		}

		// Leave room
		m.chatService.LeaveRoom(roomID, m.user.ID)

		// Remove from lists
		leftRoomName := m.rooms[roomIndex].Name
		m.rooms = append(m.rooms[:roomIndex], m.rooms[roomIndex+1:]...)
		delete(m.roomMessages, roomID)
		delete(m.roomViewports, roomID)

		// Switch to another room if needed
		if len(m.rooms) > 0 {
			if m.tabIndex >= len(m.rooms) {
				m.tabIndex = len(m.rooms) - 1
			}
			m.activeRoomID = m.rooms[m.tabIndex].ID
			m.addSystemMessage("Left " + leftRoomName)
		} else {
			m.activeRoomID = uuid.Nil
		}

	case "/create":
		if len(args) < 1 {
			m.addSystemMessage("Usage: /create <room> [--private|--password <pass>]")
			return nil
		}
		roomName := args[0]
		roomType := "public"
		password := ""

		// Parse flags
		for i := 1; i < len(args); i++ {
			if args[i] == "--private" {
				roomType = "private"
			} else if args[i] == "--password" && i+1 < len(args) {
				roomType = "password"
				password = args[i+1]
				i++ // skip the password value
			}
		}

		// Check if room already exists
		existingRoom, _ := m.chatService.GetRoomByName(roomName)
		if existingRoom != nil {
			m.addSystemMessage("Room " + roomName + " already exists")
			return nil
		}

		// Create room
		room, err := m.chatService.CreateRoom(roomName, roomType, password, m.user.ID)
		if err != nil {
			m.addSystemMessage("Error creating room: " + err.Error())
			return nil
		}

		// Auto-join the room we created
		m.chatService.JoinRoom(room.ID, m.user.ID, password)

		// Add to rooms list
		m.rooms = append(m.rooms, room)
		m.roomMessages[room.ID] = make([]models.ChatMessage, 0)
		vp := viewport.New(m.width, m.height-5)
		m.roomViewports[room.ID] = &vp
		m.updateViewportContent(room.ID)

		// Switch to new room
		m.tabIndex = len(m.rooms) - 1
		m.activeRoomID = room.ID

		typeStr := roomType
		if roomType == "password" {
			typeStr = "password-protected"
		}
		m.addSystemMessage("Created " + typeStr + " room: " + roomName)

	case "/invite":
		if len(args) < 1 {
			m.addSystemMessage("Usage: /invite <username> [room]")
			return nil
		}
		username := args[0]

		// Use current room or specified room
		var targetRoom *models.ChatRoom
		if len(args) >= 2 {
			roomName := args[1]
			for _, r := range m.rooms {
				if r.Name == roomName {
					targetRoom = r
					break
				}
			}
			if targetRoom == nil {
				m.addSystemMessage("You're not in room: " + roomName)
				return nil
			}
		} else {
			// Use current room
			if m.tabIndex >= 0 && m.tabIndex < len(m.rooms) {
				targetRoom = m.rooms[m.tabIndex]
			}
		}

		if targetRoom == nil {
			m.addSystemMessage("No room selected")
			return nil
		}

		// Look up user
		invitee, err := m.chatService.GetUserByUsername(username)
		if err != nil {
			m.addSystemMessage("User not found: " + username)
			return nil
		}

		// Invite user
		err = m.chatService.InviteUser(targetRoom.ID, m.user.ID, invitee.ID, m.user.Username)
		if err != nil {
			m.addSystemMessage("Cannot invite: " + err.Error())
			return nil
		}

		m.addSystemMessage("Invited " + username + " to " + targetRoom.Name)

	case "/exit", "/quit":
		// Exit chat mode
		m.chatService.UnregisterSession(m.sessionID)
		return func() tea.Msg {
			return chatExitMsg{}
		}

	case "/help":
		// Show help - add as a system message to current room
		helpText := `Chat Commands:
  /join <room> [password]    - Join an existing room
  /create <room> [options]   - Create room (--private or --password <pass>)
  /invite <user> [room]      - Invite user to room
  /leave [room]              - Leave room (current if no arg)
  /rooms                     - List your rooms
  /who                       - List users in current room
  /exit                      - Exit chat mode
  
Navigation: ←/→ to switch rooms, ↑/↓ for command history, Tab to autocomplete, Esc to exit`

		m.addSystemMessage(helpText)

	case "/rooms":
		// List rooms user is in
		var roomList strings.Builder
		roomList.WriteString("Your rooms: ")
		for i, room := range m.rooms {
			if i > 0 {
				roomList.WriteString(", ")
			}
			roomList.WriteString(room.Name)
		}
		m.addSystemMessage(roomList.String())

	case "/who":
		// List users in current room
		if m.activeRoomID == uuid.Nil {
			return nil
		}
		members, err := m.chatService.GetRoomMembers(m.activeRoomID)
		var content string
		if err != nil {
			content = "Error getting room members"
		} else if len(members) == 0 {
			content = "No users in this room"
		} else {
			var names []string
			for _, member := range members {
				names = append(names, member.Username)
			}
			content = "Users in room: " + strings.Join(names, ", ")
		}
		m.addSystemMessage(content)

	default:
		// Unknown command
		m.addSystemMessage("Unknown command: " + cmd + " (type /help for available commands)")
	}

	return nil
}

// addSystemMessage adds a system message to the current room
func (m *ChatModel) addSystemMessage(content string) {
	if m.activeRoomID == uuid.Nil {
		return
	}
	sysMsg := models.ChatMessage{
		ID:        uuid.New(),
		RoomID:    m.activeRoomID,
		Username:  "system",
		Content:   content,
		CreatedAt: time.Now(),
	}
	m.roomMessages[m.activeRoomID] = append(m.roomMessages[m.activeRoomID], sysMsg)
	m.updateViewportContent(m.activeRoomID)
}

// chatCommands lists all available chat commands for autocomplete
var chatCommands = []string{
	"/create",
	"/exit",
	"/help",
	"/invite",
	"/join",
	"/leave",
	"/quit",
	"/rooms",
	"/who",
}

// autocompleteCommand attempts to autocomplete a command
func (m *ChatModel) autocompleteCommand(input string) string {
	// Split input to get the command part
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input
	}

	cmdPart := parts[0]

	// If it's a complete command, try to autocomplete room names for certain commands
	if len(parts) >= 1 {
		for _, cmd := range chatCommands {
			if cmd == cmdPart {
				// Command is complete, try to autocomplete arguments
				return m.autocompleteArgs(input, cmdPart, parts[1:])
			}
		}
	}

	// Try to autocomplete the command itself
	if completed, ok := CompleteFromList(cmdPart, chatCommands); ok {
		if len(parts) > 1 {
			return completed + " " + strings.Join(parts[1:], " ")
		}
		// Add space after completed command if it was a single match
		matches := FilterByPrefix(chatCommands, cmdPart)
		if len(matches) == 1 {
			return completed + " "
		}
		return completed
	}

	return input
}

// autocompleteArgs attempts to autocomplete command arguments (room names, usernames)
func (m *ChatModel) autocompleteArgs(input, cmd string, args []string) string {
	switch cmd {
	case "/join", "/leave":
		// Autocomplete room names
		if len(args) == 0 || (len(args) == 1 && !strings.HasSuffix(input, " ")) {
			partial := ""
			if len(args) == 1 {
				partial = args[0]
			}

			// Get room names
			var roomNames []string
			for _, room := range m.rooms {
				roomNames = append(roomNames, room.Name)
			}

			if completed, ok := CompleteFromList(partial, roomNames); ok {
				return cmd + " " + completed
			}
		}
	case "/invite":
		// Could autocomplete usernames if we had a list, for now just return as-is
		// Future: add GetOnlineUsers() method to ChatService
	}

	return input
}

// updateViewportContent updates the viewport content for a room
func (m *ChatModel) updateViewportContent(roomID uuid.UUID) {
	vp, ok := m.roomViewports[roomID]
	if !ok {
		return
	}

	messages := m.roomMessages[roomID]
	var content strings.Builder

	for _, msg := range messages {
		timestamp := msg.CreatedAt.Format("15:04:05") // Uses time package
		line := fmt.Sprintf("[%s] <%s> %s", timestamp, msg.Username, msg.Content)
		content.WriteString(line)
		content.WriteString("\n")
	}

	vp.SetContent(content.String())
	vp.GotoBottom()
}

// View renders the chat UI
func (m *ChatModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Add top padding for web
	sb.WriteString("\n")

	// Render tabs
	if len(m.rooms) > 0 {
		tabStyle := lipgloss.NewStyle().Padding(0, 1).MarginRight(1)
		activeTabStyle := tabStyle.Copy().Bold(true).Foreground(lipgloss.Color("205"))
		inactiveTabStyle := tabStyle.Copy().Foreground(lipgloss.Color("240"))

		for i, room := range m.rooms {
			if i == m.tabIndex {
				sb.WriteString(activeTabStyle.Render(room.Name))
			} else {
				sb.WriteString(inactiveTabStyle.Render(room.Name))
			}
		}
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", m.width))
		sb.WriteString("\n")

		// Render viewport
		if vp, ok := m.roomViewports[m.activeRoomID]; ok {
			sb.WriteString(vp.View())
		}
	} else {
		// No rooms - pad to push content to bottom
		// Calculate lines needed: height - top padding (1) - message line (1) - input line (1) - buffer (1)
		emptyLines := m.height - 4
		if emptyLines < 0 {
			emptyLines = 0
		}
		for i := 0; i < emptyLines; i++ {
			sb.WriteString("\n")
		}
		sb.WriteString("No rooms. Use /create <room> to create one, or /join <room> to join.")
	}

	// Input on separate line
	sb.WriteString("\n")
	sb.WriteString(m.textInput.View())

	return sb.String()
}

// Close cleans up the chat model
func (m *ChatModel) Close() {
	m.chatService.UnregisterSession(m.sessionID)
}

// StartMessageLoop starts the message listener and forwards messages to the given channel
// This is used by the WebSocket bridge since it can't run Init() commands properly
func (m *ChatModel) StartMessageLoop(bridgeChan chan tea.Msg) {
	go func() {
		for {
			msg, ok := <-m.msgChan
			if !ok {
				return // Channel closed
			}
			// Forward to bridge channel as ChatMessageMsg
			select {
			case bridgeChan <- ChatMessageMsg{Message: msg}:
			default:
				// Bridge channel full or closed, stop listening
				return
			}
		}
	}()
}

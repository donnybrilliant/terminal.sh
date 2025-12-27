package cmd

import (
	"fmt"
	"strings"

	"terminal-sh/models"

	"github.com/google/uuid"
)

// handleChat handles the chat command
func (h *CommandHandler) handleChat(args []string) *CommandResult {
	if h.user == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if h.chatService == nil {
		return &CommandResult{Error: fmt.Errorf("chat service not available")}
	}

	// Check for flags
	splitMode := false
	if len(args) > 0 {
		for _, arg := range args {
			if arg == "--split" || arg == "-s" {
				splitMode = true
			}
		}
	}

	// Return special marker to enter chat mode
	if splitMode {
		return &CommandResult{Output: "__CHAT_MODE_SPLIT__"}
	}
	return &CommandResult{Output: "__CHAT_MODE__"}
}

// handleChatJoin handles /join command in chat mode
func (h *CommandHandler) handleChatJoin(args []string) *CommandResult {
	if h.user == nil || h.chatService == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) < 1 {
		return &CommandResult{Error: fmt.Errorf("usage: /join <room> [password]")}
	}

	roomName := args[0]
	password := ""
	if len(args) > 1 {
		password = args[1]
	}

	// Get or create room
	room, err := h.chatService.GetRoomByName(roomName)
	if err != nil {
		// Room doesn't exist, create it as public
		room, err = h.chatService.CreateRoom(roomName, "public", "", h.user.ID)
		if err != nil {
			return &CommandResult{Error: fmt.Errorf("failed to create room: %w", err)}
		}
	}

	// Join room
	err = h.chatService.JoinRoom(room.ID, h.user.ID, password)
	if err != nil {
		return &CommandResult{Error: err}
	}

	return &CommandResult{Output: fmt.Sprintf("Joined room: %s\n", roomName)}
}

// handleChatLeave handles /leave command in chat mode
func (h *CommandHandler) handleChatLeave(args []string) *CommandResult {
	if h.user == nil || h.chatService == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) < 1 {
		return &CommandResult{Error: fmt.Errorf("usage: /leave <room>")}
	}

	roomName := args[0]

	// Get room
	room, err := h.chatService.GetRoomByName(roomName)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("room not found: %s", roomName)}
	}

	// Leave room
	err = h.chatService.LeaveRoom(room.ID, h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	return &CommandResult{Output: fmt.Sprintf("Left room: %s\n", roomName)}
}

// handleChatCreate handles /create command in chat mode
func (h *CommandHandler) handleChatCreate(args []string) *CommandResult {
	if h.user == nil || h.chatService == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) < 1 {
		return &CommandResult{Error: fmt.Errorf("usage: /create <room> [--private|--password <pass>]")}
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
			i++ // Skip password argument
		}
	}

	// Create room
	room, err := h.chatService.CreateRoom(roomName, roomType, password, h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Auto-join creator
	err = h.chatService.JoinRoom(room.ID, h.user.ID, password)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("failed to join created room: %w", err)}
	}

	return &CommandResult{Output: fmt.Sprintf("Created and joined room: %s\n", roomName)}
}

// handleChatRooms handles /rooms command in chat mode
func (h *CommandHandler) handleChatRooms() *CommandResult {
	if h.user == nil || h.chatService == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	rooms, err := h.chatService.GetRoomsForUser(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	if len(rooms) == 0 {
		return &CommandResult{Output: "You are not in any rooms.\n"}
	}

	var output strings.Builder
	output.WriteString("Rooms you are in:\n")
	for _, room := range rooms {
		output.WriteString(fmt.Sprintf("  - %s (%s)\n", room.Name, room.Type))
	}

	return &CommandResult{Output: output.String()}
}

// handleChatWho handles /who command in chat mode
func (h *CommandHandler) handleChatWho(args []string) *CommandResult {
	if h.user == nil || h.chatService == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) < 1 {
		return &CommandResult{Error: fmt.Errorf("usage: /who <room>")}
	}

	roomName := args[0]

	// Get room
	room, err := h.chatService.GetRoomByName(roomName)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("room not found: %s", roomName)}
	}

	// Get members
	memberIDs, err := h.chatService.GetRoomMembers(room.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	// Get usernames
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Users in %s:\n", roomName))
	for _, userID := range memberIDs {
		var user models.User
		if err := h.db.First(&user, "id = ?", userID).Error; err == nil {
			output.WriteString(fmt.Sprintf("  - %s\n", user.Username))
		}
	}

	return &CommandResult{Output: output.String()}
}

// handleChatInvite handles /invite command in chat mode
func (h *CommandHandler) handleChatInvite(args []string) *CommandResult {
	if h.user == nil || h.chatService == nil {
		return &CommandResult{Error: fmt.Errorf("not authenticated")}
	}

	if len(args) < 2 {
		return &CommandResult{Error: fmt.Errorf("usage: /invite <user> <room>")}
	}

	username := args[0]
	roomName := args[1]

	// Get user
	var targetUser models.User
	if err := h.db.Where("username = ?", username).First(&targetUser).Error; err != nil {
		return &CommandResult{Error: fmt.Errorf("user not found: %s", username)}
	}

	// Get room
	room, err := h.chatService.GetRoomByName(roomName)
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("room not found: %s", roomName)}
	}

	// Check if room is private
	if room.Type != "private" {
		return &CommandResult{Error: fmt.Errorf("room is not private, use /join instead")}
	}

	// Check if current user is in the room
	rooms, err := h.chatService.GetRoomsForUser(h.user.ID)
	if err != nil {
		return &CommandResult{Error: err}
	}

	isMember := false
	for _, r := range rooms {
		if r.ID == room.ID {
			isMember = true
			break
		}
	}

	if !isMember {
		return &CommandResult{Error: fmt.Errorf("you must be a member of the room to invite others")}
	}

	// Add user to room (bypass password check for private rooms)
	err = h.chatService.JoinRoom(room.ID, targetUser.ID, "")
	if err != nil {
		return &CommandResult{Error: fmt.Errorf("failed to invite user: %w", err)}
	}

	return &CommandResult{Output: fmt.Sprintf("Invited %s to %s\n", username, roomName)}
}

// handleChatMessage handles sending a message in chat mode
func (h *CommandHandler) handleChatMessage(roomID uuid.UUID, content string) error {
	if h.user == nil || h.chatService == nil {
		return fmt.Errorf("not authenticated")
	}

	return h.chatService.SendMessage(roomID, h.user.ID, h.user.Username, content)
}


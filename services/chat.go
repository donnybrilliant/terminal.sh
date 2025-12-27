package services

import (
	"fmt"
	"sync"
	"time"

	"terminal-sh/auth"
	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// ChatService manages chat rooms, memberships, and message broadcasting
type ChatService struct {
	db             *database.Database
	rooms          map[uuid.UUID]*models.ChatRoom
	activeSessions map[uuid.UUID]chan models.ChatMessage
	sessionUsers   map[uuid.UUID]uuid.UUID          // sessionID -> userID
	roomMembers    map[uuid.UUID]map[uuid.UUID]bool // roomID -> userID -> bool
	mu             sync.RWMutex
}

// NewChatService creates a new chat service
func NewChatService(db *database.Database) *ChatService {
	service := &ChatService{
		db:             db,
		rooms:          make(map[uuid.UUID]*models.ChatRoom),
		activeSessions: make(map[uuid.UUID]chan models.ChatMessage),
		sessionUsers:   make(map[uuid.UUID]uuid.UUID),
		roomMembers:    make(map[uuid.UUID]map[uuid.UUID]bool),
	}

	// Load rooms from database into memory
	service.loadRooms()

	return service
}

// loadRooms loads all rooms from database into memory cache
func (s *ChatService) loadRooms() {
	var rooms []models.ChatRoom
	if err := s.db.Find(&rooms).Error; err == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i := range rooms {
			s.rooms[rooms[i].ID] = &rooms[i]
			// Load members for each room
			s.loadRoomMembers(rooms[i].ID)
		}
	}
}

// loadRoomMembers loads members for a room into memory
func (s *ChatService) loadRoomMembers(roomID uuid.UUID) {
	var members []models.ChatRoomMember
	if err := s.db.Where("room_id = ?", roomID).Find(&members).Error; err == nil {
		if s.roomMembers[roomID] == nil {
			s.roomMembers[roomID] = make(map[uuid.UUID]bool)
		}
		for _, member := range members {
			s.roomMembers[roomID][member.UserID] = true
		}
	}
}

// InitializeDefaultRoom creates the default #public room if it doesn't exist
func (s *ChatService) InitializeDefaultRoom() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if #public room already exists
	var existingRoom models.ChatRoom
	if err := s.db.Where("name = ?", "#public").First(&existingRoom).Error; err == nil {
		// Room exists, add to cache if not already there
		if s.rooms[existingRoom.ID] == nil {
			s.rooms[existingRoom.ID] = &existingRoom
			s.loadRoomMembers(existingRoom.ID)
		}
		return nil
	}

	// Create default room (no creator, system room)
	defaultUserID := uuid.Nil // System room
	room := &models.ChatRoom{
		Name:      "#public",
		Type:      "public",
		Password:  "",
		CreatedBy: defaultUserID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(room).Error; err != nil {
		return fmt.Errorf("failed to create default room: %w", err)
	}

	s.rooms[room.ID] = room
	s.roomMembers[room.ID] = make(map[uuid.UUID]bool)

	return nil
}

// CreateRoom creates a new chat room
func (s *ChatService) CreateRoom(name, roomType, password string, creatorID uuid.UUID) (*models.ChatRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate room type
	if roomType != "public" && roomType != "private" && roomType != "password" {
		return nil, fmt.Errorf("invalid room type: %s", roomType)
	}

	// Check if room already exists
	var existingRoom models.ChatRoom
	if err := s.db.Where("name = ?", name).First(&existingRoom).Error; err == nil {
		return nil, fmt.Errorf("room already exists: %s", name)
	}

	// Hash password if provided
	hashedPassword := ""
	if roomType == "password" && password != "" {
		hash, err := auth.HashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		hashedPassword = hash
	}

	room := &models.ChatRoom{
		Name:      name,
		Type:      roomType,
		Password:  hashedPassword,
		CreatedBy: creatorID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(room).Error; err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	s.rooms[room.ID] = room
	s.roomMembers[room.ID] = make(map[uuid.UUID]bool)

	return room, nil
}

// GetRoomByName retrieves a room by name
func (s *ChatService) GetRoomByName(name string) (*models.ChatRoom, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check cache first
	for _, room := range s.rooms {
		if room.Name == name {
			return room, nil
		}
	}

	// Check database
	var room models.ChatRoom
	if err := s.db.Where("name = ?", name).First(&room).Error; err != nil {
		return nil, err
	}

	// Add to cache
	s.mu.RUnlock()
	s.mu.Lock()
	s.rooms[room.ID] = &room
	s.loadRoomMembers(room.ID)
	s.mu.Unlock()
	s.mu.RLock()

	return &room, nil
}

// JoinRoom adds a user to a room
func (s *ChatService) JoinRoom(roomID, userID uuid.UUID, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get room
	room, ok := s.rooms[roomID]
	if !ok {
		// Try to load from database
		var dbRoom models.ChatRoom
		if err := s.db.First(&dbRoom, "id = ?", roomID).Error; err != nil {
			return fmt.Errorf("room not found")
		}
		room = &dbRoom
		s.rooms[roomID] = room
		s.loadRoomMembers(roomID)
	}

	// Check if already a member
	if s.roomMembers[roomID] != nil && s.roomMembers[roomID][userID] {
		return nil // Already a member
	}

	// Check room type and password
	if room.Type == "private" {
		return fmt.Errorf("private room requires invitation")
	}
	if room.Type == "password" {
		if password == "" {
			return fmt.Errorf("password required")
		}
		if !auth.CheckPasswordHash(password, room.Password) {
			return fmt.Errorf("incorrect password")
		}
	}

	// Add membership
	member := &models.ChatRoomMember{
		RoomID:   roomID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	if err := s.db.Create(member).Error; err != nil {
		return fmt.Errorf("failed to join room: %w", err)
	}

	if s.roomMembers[roomID] == nil {
		s.roomMembers[roomID] = make(map[uuid.UUID]bool)
	}
	s.roomMembers[roomID][userID] = true

	return nil
}

// LeaveRoom removes a user from a room
func (s *ChatService) LeaveRoom(roomID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from database
	if err := s.db.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&models.ChatRoomMember{}).Error; err != nil {
		return fmt.Errorf("failed to leave room: %w", err)
	}

	// Remove from memory
	if s.roomMembers[roomID] != nil {
		delete(s.roomMembers[roomID], userID)
	}

	return nil
}

// SendMessage sends a message to a room
func (s *ChatService) SendMessage(roomID, userID uuid.UUID, username, content string) error {
	s.mu.RLock()

	// Check if user is member of room
	if s.roomMembers[roomID] == nil || !s.roomMembers[roomID][userID] {
		s.mu.RUnlock()
		return fmt.Errorf("user is not a member of this room")
	}

	s.mu.RUnlock()

	// Create message
	message := &models.ChatMessage{
		RoomID:    roomID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(message).Error; err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Trim old messages (keep last 100)
	s.trimMessages(roomID)

	// Broadcast to all active sessions in the room
	s.broadcastMessage(*message, roomID)

	return nil
}

// trimMessages keeps only the last 100 messages for a room
func (s *ChatService) trimMessages(roomID uuid.UUID) {
	var count int64
	s.db.Model(&models.ChatMessage{}).Where("room_id = ?", roomID).Count(&count)

	if count > 100 {
		// Get the 100th oldest message's ID
		var messages []models.ChatMessage
		s.db.Where("room_id = ?", roomID).
			Order("created_at ASC").
			Limit(1).
			Offset(99).
			Find(&messages)

		if len(messages) > 0 {
			// Delete all messages older than this
			s.db.Where("room_id = ? AND created_at < ?", roomID, messages[0].CreatedAt).
				Delete(&models.ChatMessage{})
		}
	}
}

// broadcastMessage sends a message to all active sessions in a room
func (s *ChatService) broadcastMessage(message models.ChatMessage, roomID uuid.UUID) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all user IDs in the room
	userIDs := make(map[uuid.UUID]bool)
	if s.roomMembers[roomID] != nil {
		for userID := range s.roomMembers[roomID] {
			userIDs[userID] = true
		}
	}

	// Send to all active sessions for users in the room
	for sessionID, msgChan := range s.activeSessions {
		// Check if this session's user is in the room
		if userID, ok := s.sessionUsers[sessionID]; ok && userIDs[userID] {
			select {
			case msgChan <- message:
			default:
				// Channel full, skip this session
			}
		}
	}
}

// RegisterSession registers an active session and returns a message channel
func (s *ChatService) RegisterSession(sessionID, userID uuid.UUID) chan models.ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create buffered channel for messages
	msgChan := make(chan models.ChatMessage, 100)
	s.activeSessions[sessionID] = msgChan
	s.sessionUsers[sessionID] = userID

	return msgChan
}

// UnregisterSession removes an active session
func (s *ChatService) UnregisterSession(sessionID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msgChan, ok := s.activeSessions[sessionID]; ok {
		close(msgChan)
		delete(s.activeSessions, sessionID)
		delete(s.sessionUsers, sessionID)
	}
}

// GetRoomsForUser returns all rooms a user is a member of
func (s *ChatService) GetRoomsForUser(userID uuid.UUID) ([]*models.ChatRoom, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var members []models.ChatRoomMember
	if err := s.db.Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to get user rooms: %w", err)
	}

	rooms := make([]*models.ChatRoom, 0, len(members))
	for _, member := range members {
		if room, ok := s.rooms[member.RoomID]; ok {
			rooms = append(rooms, room)
		} else {
			// Load from database if not in cache
			var dbRoom models.ChatRoom
			if err := s.db.First(&dbRoom, "id = ?", member.RoomID).Error; err == nil {
				s.mu.RUnlock()
				s.mu.Lock()
				s.rooms[dbRoom.ID] = &dbRoom
				s.loadRoomMembers(dbRoom.ID)
				s.mu.Unlock()
				s.mu.RLock()
				rooms = append(rooms, &dbRoom)
			}
		}
	}

	return rooms, nil
}

// GetRecentMessages returns recent messages for a room
func (s *ChatService) GetRecentMessages(roomID uuid.UUID, limit int) ([]models.ChatMessage, error) {
	if limit <= 0 {
		limit = 100
	}

	var messages []models.ChatMessage
	if err := s.db.Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetRoomMembers returns all users in a room with their usernames
func (s *ChatService) GetRoomMembers(roomID uuid.UUID) ([]models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var members []models.ChatRoomMember
	if err := s.db.Where("room_id = ?", roomID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}

	userIDs := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}

	// Look up usernames
	var users []models.User
	if len(userIDs) > 0 {
		if err := s.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}
	}

	return users, nil
}

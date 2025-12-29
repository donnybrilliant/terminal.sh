package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"terminal-sh/database"
	"terminal-sh/services"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now (can be restricted in production)
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocketSession manages a WebSocket connection and its associated Bubble Tea program.
type WebSocketSession struct {
	conn      *websocket.Conn
	bridge    *BubbleTeaBridge
	db        *database.Database
	userService *services.UserService
	width     int
	height    int
}

// HandleWebSocket handles WebSocket upgrade and manages the session lifecycle.
// Creates a Bubble Tea bridge and processes incoming messages until the connection closes.
func HandleWebSocket(w http.ResponseWriter, r *http.Request, db *database.Database, userService *services.UserService, chatService *services.ChatService) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Default terminal size (will be updated on first resize message)
	width := 80
	height := 24

	// Create Bubble Tea bridge
	bridge, err := NewBubbleTeaBridge(conn, db, userService, chatService, width, height)
	if err != nil {
		return err
	}
	defer bridge.Close()

	session := &WebSocketSession{
		conn:        conn,
		bridge:      bridge,
		db:          db,
		userService: userService,
		width:       width,
		height:      height,
	}

	// Handle messages from client
	return session.handleMessages()
}

// handleMessages processes incoming WebSocket messages
func (s *WebSocketSession) handleMessages() error {
	for {
		// Read message from client
		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return err
		}

		if messageType != websocket.TextMessage {
			continue
		}

		// Parse message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		// Handle different message types
		switch msgType {
		case MessageTypeInput:
			var inputMsg InputMessage
			if err := json.Unmarshal(message, &inputMsg); err == nil {
				if err := s.bridge.HandleInput(inputMsg); err != nil {
					log.Printf("Error handling input: %v", err)
				}
			}
		case MessageTypeResize:
			var resizeMsg ResizeMessage
			if err := json.Unmarshal(message, &resizeMsg); err == nil {
				s.width = resizeMsg.Width
				s.height = resizeMsg.Height
				if err := s.bridge.HandleResize(resizeMsg); err != nil {
					log.Printf("Error handling resize: %v", err)
				}
			}
		case MessageTypeMouse:
			var mouseMsg MouseMessage
			if err := json.Unmarshal(message, &mouseMsg); err == nil {
				if err := s.bridge.HandleMouse(mouseMsg); err != nil {
					log.Printf("Error handling mouse: %v", err)
				}
			}
		case MessageTypeClose:
			return nil // Client requested close
		}
	}
}


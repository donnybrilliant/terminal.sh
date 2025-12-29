package services

import (
	"fmt"
	"time"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// SessionService handles SSH and terminal session management.
type SessionService struct {
	db            *database.Database
	serverService *ServerService
}

// NewSessionService creates a new SessionService with the provided database and server service.
func NewSessionService(db *database.Database, serverService *ServerService) *SessionService {
	return &SessionService{
		db:            db,
		serverService: serverService,
	}
}

// CreateSession creates a new SSH or terminal session for a user.
// Supports nested sessions (SSH within SSH) via parentSessionID.
func (s *SessionService) CreateSession(userID uuid.UUID, sshConnID string, currentServerPath string, parentSessionID *uuid.UUID) (*models.Session, error) {
	session := &models.Session{
		UserID:          userID,
		SSHConnID:       sshConnID,
		CurrentServerPath: currentServerPath,
		ParentSessionID: parentSessionID,
		CreatedAt:       time.Now(),
	}

	if err := s.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSessionByConnID retrieves a session by SSH connection ID
func (s *SessionService) GetSessionByConnID(sshConnID string) (*models.Session, error) {
	var session models.Session
	if err := s.db.Where("ssh_conn_id = ?", sshConnID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateSessionServerPath updates the current server path for a session
func (s *SessionService) UpdateSessionServerPath(sessionID uuid.UUID, serverPath string) error {
	return s.db.Model(&models.Session{}).Where("id = ?", sessionID).Update("current_server_path", serverPath).Error
}

// GetSessionHierarchy returns the full session hierarchy (parent sessions)
func (s *SessionService) GetSessionHierarchy(sessionID uuid.UUID) ([]*models.Session, error) {
	var sessions []*models.Session
	currentSessionID := sessionID

	for currentSessionID != uuid.Nil {
		var session models.Session
		if err := s.db.Where("id = ?", currentSessionID).First(&session).Error; err != nil {
			break
		}
		sessions = append([]*models.Session{&session}, sessions...)
		
		if session.ParentSessionID == nil {
			break
		}
		currentSessionID = *session.ParentSessionID
	}

	return sessions, nil
}

// BuildServerPath builds a server path from session hierarchy
func (s *SessionService) BuildServerPath(sessions []*models.Session) string {
	if len(sessions) == 0 {
		return ""
	}

	path := sessions[0].CurrentServerPath
	for i := 1; i < len(sessions); i++ {
		if sessions[i].CurrentServerPath != "" {
			path += ".localNetwork." + sessions[i].CurrentServerPath
		}
	}

	return path
}

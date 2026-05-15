package services

import (
	"strings"
	"time"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

// ServerLogService handles server log operations.
type ServerLogService struct {
	db *database.Database
}

// NewServerLogService creates a new ServerLogService.
func NewServerLogService(db *database.Database) *ServerLogService {
	return &ServerLogService{db: db}
}

// LogConnect logs a connection to a server via any service type.
func (s *ServerLogService) LogConnect(serverIP, sourceIP, username string, userID *uuid.UUID, serviceType string, success bool) error {
	log := &models.ServerLog{
		ServerIP:    serverIP,
		LogType:     models.LogTypeConnect,
		ServiceType: serviceType,
		SourceIP:    sourceIP,
		Username:    username,
		UserID:      userID,
		Message:     serviceType + " connection",
		Success:     success,
		CreatedAt:   time.Now(),
	}
	return s.db.Create(log).Error
}

// LogDisconnect logs a disconnection from a server.
func (s *ServerLogService) LogDisconnect(serverIP, sourceIP, username string, userID *uuid.UUID, serviceType string) error {
	log := &models.ServerLog{
		ServerIP:    serverIP,
		LogType:     models.LogTypeDisconnect,
		ServiceType: serviceType,
		SourceIP:    sourceIP,
		Username:    username,
		UserID:      userID,
		Message:     serviceType + " disconnection",
		Success:     true,
		CreatedAt:   time.Now(),
	}
	return s.db.Create(log).Error
}

// LogSSHConnect logs an SSH connection to a server.
// Deprecated: Use LogConnect with serviceType="ssh" instead.
func (s *ServerLogService) LogSSHConnect(serverIP, sourceIP, username string, userID *uuid.UUID, success bool) error {
	return s.LogConnect(serverIP, sourceIP, username, userID, "ssh", success)
}

// LogSSHDisconnect logs an SSH disconnection from a server.
// Deprecated: Use LogDisconnect with serviceType="ssh" instead.
func (s *ServerLogService) LogSSHDisconnect(serverIP, sourceIP, username string, userID *uuid.UUID) error {
	return s.LogDisconnect(serverIP, sourceIP, username, userID, "ssh")
}

// LogExploitAttempt logs an exploitation attempt on a server.
func (s *ServerLogService) LogExploitAttempt(serverIP, sourceIP, username string, userID *uuid.UUID, toolName, serviceName string, success bool) error {
	logType := models.LogTypeExploitSuccess
	if !success {
		logType = models.LogTypeExploitFail
	}

	log := &models.ServerLog{
		ServerIP:  serverIP,
		LogType:   logType,
		SourceIP:  sourceIP,
		Username:  username,
		UserID:    userID,
		Message:   "Exploitation attempt",
		Details:   toolName + " on " + serviceName,
		Success:   success,
		CreatedAt: time.Now(),
	}
	return s.db.Create(log).Error
}

// LogCommand logs a command executed on a server.
func (s *ServerLogService) LogCommand(serverIP, sourceIP, username string, userID *uuid.UUID, command string, success bool) error {
	log := &models.ServerLog{
		ServerIP:  serverIP,
		LogType:   models.LogTypeCommand,
		SourceIP:  sourceIP,
		Username:  username,
		UserID:    userID,
		Message:   "Command executed",
		Details:   command,
		Success:   success,
		CreatedAt: time.Now(),
	}
	return s.db.Create(log).Error
}

// LogFileRead logs a file read operation on a server.
func (s *ServerLogService) LogFileRead(serverIP, username string, userID *uuid.UUID, filePath string) error {
	log := &models.ServerLog{
		ServerIP:  serverIP,
		LogType:   models.LogTypeFileRead,
		Username:  username,
		UserID:    userID,
		Message:   "File read",
		Details:   filePath,
		Success:   true,
		CreatedAt: time.Now(),
	}
	return s.db.Create(log).Error
}

// LogFileWrite logs a file write operation on a server.
func (s *ServerLogService) LogFileWrite(serverIP, username string, userID *uuid.UUID, filePath string) error {
	log := &models.ServerLog{
		ServerIP:  serverIP,
		LogType:   models.LogTypeFileWrite,
		Username:  username,
		UserID:    userID,
		Message:   "File write",
		Details:   filePath,
		Success:   true,
		CreatedAt: time.Now(),
	}
	return s.db.Create(log).Error
}

// LogScan logs a port scan detected on a server.
func (s *ServerLogService) LogScan(serverIP, sourceIP string, userID *uuid.UUID) error {
	log := &models.ServerLog{
		ServerIP:  serverIP,
		LogType:   models.LogTypeScan,
		SourceIP:  sourceIP,
		UserID:    userID,
		Message:   "Port scan detected",
		Success:   true,
		CreatedAt: time.Now(),
	}
	return s.db.Create(log).Error
}

// LogSystem logs a system event on a server.
func (s *ServerLogService) LogSystem(serverIP, message string) error {
	log := &models.ServerLog{
		ServerIP:  serverIP,
		LogType:   models.LogTypeSystem,
		Message:   message,
		Success:   true,
		CreatedAt: time.Now(),
	}
	return s.db.Create(log).Error
}

// GetServerLogs retrieves logs for a server, optionally filtered by type.
func (s *ServerLogService) GetServerLogs(serverIP string, logType *models.LogType, limit int) ([]models.ServerLog, error) {
	var logs []models.ServerLog
	query := s.db.Where("server_ip = ?", serverIP)
	if logType != nil {
		query = query.Where("log_type = ?", *logType)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("created_at DESC").Find(&logs).Error
	return logs, err
}

// GetAuthLogs retrieves auth-related logs for a server (connections, exploits).
func (s *ServerLogService) GetAuthLogs(serverIP string, limit int) ([]models.ServerLog, error) {
	var logs []models.ServerLog
	query := s.db.Where("server_ip = ? AND log_type IN ?", serverIP, []models.LogType{
		models.LogTypeConnect,
		models.LogTypeDisconnect,
		models.LogTypeSSHConnect,    // backward compatibility
		models.LogTypeSSHDisconnect, // backward compatibility
		models.LogTypeExploitAttempt,
		models.LogTypeExploitSuccess,
		models.LogTypeExploitFail,
		models.LogTypeAuth,
	})
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("created_at DESC").Find(&logs).Error
	return logs, err
}

// GetSystemLogs retrieves system-related logs for a server (commands, file ops).
func (s *ServerLogService) GetSystemLogs(serverIP string, limit int) ([]models.ServerLog, error) {
	var logs []models.ServerLog
	query := s.db.Where("server_ip = ? AND log_type IN ?", serverIP, []models.LogType{
		models.LogTypeCommand,
		models.LogTypeFileRead,
		models.LogTypeFileWrite,
		models.LogTypeScan,
		models.LogTypeSystem,
	})
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("created_at DESC").Find(&logs).Error
	return logs, err
}

// FormatAuthLog formats auth logs for display in /var/log/auth.log.
func (s *ServerLogService) FormatAuthLog(serverIP string, limit int) (string, error) {
	logs, err := s.GetAuthLogs(serverIP, limit)
	if err != nil {
		return "", err
	}

	// Reverse order so oldest is first (like real logs)
	var lines []string
	for i := len(logs) - 1; i >= 0; i-- {
		lines = append(lines, logs[i].FormatForAuthLog())
	}
	return strings.Join(lines, "\n"), nil
}

// FormatSystemLog formats system logs for display in /var/log/system.log.
func (s *ServerLogService) FormatSystemLog(serverIP string, limit int) (string, error) {
	logs, err := s.GetSystemLogs(serverIP, limit)
	if err != nil {
		return "", err
	}

	// Reverse order so oldest is first (like real logs)
	var lines []string
	for i := len(logs) - 1; i >= 0; i-- {
		lines = append(lines, logs[i].FormatForSystemLog())
	}
	return strings.Join(lines, "\n"), nil
}

// GetUserActivityOnServer retrieves all logs for a specific user on a server.
func (s *ServerLogService) GetUserActivityOnServer(serverIP string, userID uuid.UUID, limit int) ([]models.ServerLog, error) {
	var logs []models.ServerLog
	query := s.db.Where("server_ip = ? AND user_id = ?", serverIP, userID)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("created_at DESC").Find(&logs).Error
	return logs, err
}

// CleanOldLogs removes logs older than the specified duration.
func (s *ServerLogService) CleanOldLogs(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	return s.db.Where("created_at < ?", cutoff).Delete(&models.ServerLog{}).Error
}

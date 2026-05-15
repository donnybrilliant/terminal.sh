package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LogType represents the type of server log entry.
type LogType string

const (
	LogTypeConnect       LogType = "connect"        // Generalized connection (SSH, FTP, Telnet, etc.)
	LogTypeDisconnect    LogType = "disconnect"     // Generalized disconnection
	LogTypeSSHConnect    LogType = "ssh_connect"    // Deprecated: use LogTypeConnect with ServiceType
	LogTypeSSHDisconnect LogType = "ssh_disconnect" // Deprecated: use LogTypeDisconnect with ServiceType
	LogTypeExploitAttempt LogType = "exploit_attempt"
	LogTypeExploitSuccess LogType = "exploit_success"
	LogTypeExploitFail   LogType = "exploit_fail"
	LogTypeCommand       LogType = "command"
	LogTypeFileRead      LogType = "file_read"
	LogTypeFileWrite     LogType = "file_write"
	LogTypeScan          LogType = "scan"
	LogTypeAuth          LogType = "auth"
	LogTypeSystem        LogType = "system"
)

// ServerLog represents a log entry for a server.
// Logs are created dynamically as users interact with servers.
type ServerLog struct {
	ID          uuid.UUID  `gorm:"type:text;primary_key" json:"id"`
	ServerIP    string     `gorm:"not null;index" json:"server_ip"` // Server IP (not path, so logs are per-server)
	LogType     LogType    `gorm:"not null;index" json:"log_type"`
	ServiceType string     `json:"service_type"`                     // Service type (ssh, ftp, telnet, etc.)
	SourceIP    string     `json:"source_ip"`                        // IP of the connecting user/attacker
	Username    string     `json:"username"`                         // Username if known
	UserID      *uuid.UUID `gorm:"type:text;index" json:"user_id"`   // User ID if authenticated (nil for anonymous)
	Message     string     `gorm:"not null" json:"message"`          // Log message
	Details     string     `json:"details"`                          // Additional details (command output, tool name, etc.)
	Success     bool       `json:"success"`                          // Whether the action succeeded
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
}

// GetDaemonName returns the daemon name for logging based on service type.
func GetDaemonName(serviceType string) string {
	switch serviceType {
	case "ssh":
		return "sshd"
	case "ftp":
		return "ftpd"
	case "telnet":
		return "telnetd"
	case "rdp":
		return "rdpd"
	case "vnc":
		return "vncd"
	default:
		if serviceType != "" {
			return serviceType + "d"
		}
		return "sshd" // fallback for backward compatibility
	}
}

// BeforeCreate is a GORM hook that generates a UUID for the log if one doesn't exist.
func (l *ServerLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// FormatForAuthLog formats the log entry for /var/log/auth.log style output.
func (l *ServerLog) FormatForAuthLog() string {
	timestamp := l.CreatedAt.Format("Jan 02 15:04:05")
	daemon := GetDaemonName(l.ServiceType)
	
	switch l.LogType {
	case LogTypeConnect, LogTypeSSHConnect:
		if l.Success {
			return timestamp + " " + daemon + "[" + l.ID.String()[:8] + "]: Accepted connection from " + l.SourceIP + " user " + l.Username
		}
		return timestamp + " " + daemon + "[" + l.ID.String()[:8] + "]: Failed connection from " + l.SourceIP
	case LogTypeDisconnect, LogTypeSSHDisconnect:
		return timestamp + " " + daemon + "[" + l.ID.String()[:8] + "]: Connection closed by " + l.SourceIP + " user " + l.Username
	case LogTypeExploitAttempt:
		return timestamp + " " + daemon + "[" + l.ID.String()[:8] + "]: Exploit attempt from " + l.SourceIP + " - " + l.Details
	case LogTypeExploitSuccess:
		return timestamp + " " + daemon + "[" + l.ID.String()[:8] + "]: WARNING: Successful exploit from " + l.SourceIP + " - " + l.Details
	case LogTypeExploitFail:
		return timestamp + " " + daemon + "[" + l.ID.String()[:8] + "]: Blocked exploit from " + l.SourceIP + " - " + l.Details
	case LogTypeAuth:
		if l.Success {
			return timestamp + " auth[" + l.ID.String()[:8] + "]: Authenticated " + l.Username + " from " + l.SourceIP
		}
		return timestamp + " auth[" + l.ID.String()[:8] + "]: Authentication failed for " + l.Username + " from " + l.SourceIP
	default:
		return timestamp + " " + string(l.LogType) + ": " + l.Message
	}
}

// FormatForSystemLog formats the log entry for /var/log/system.log style output.
func (l *ServerLog) FormatForSystemLog() string {
	timestamp := l.CreatedAt.Format("Jan 02 15:04:05")
	
	switch l.LogType {
	case LogTypeCommand:
		return timestamp + " shell[" + l.ID.String()[:8] + "]: " + l.Username + " executed: " + l.Details
	case LogTypeFileRead:
		return timestamp + " fs[" + l.ID.String()[:8] + "]: " + l.Username + " read file: " + l.Details
	case LogTypeFileWrite:
		return timestamp + " fs[" + l.ID.String()[:8] + "]: " + l.Username + " wrote file: " + l.Details
	case LogTypeScan:
		return timestamp + " net[" + l.ID.String()[:8] + "]: Port scan detected from " + l.SourceIP
	case LogTypeSystem:
		return timestamp + " system: " + l.Message
	default:
		return timestamp + " " + string(l.LogType) + ": " + l.Message
	}
}

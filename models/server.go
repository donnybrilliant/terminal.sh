package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServerResources represents server computational resources.
type ServerResources struct {
	CPU      int     `json:"cpu"`
	Bandwidth float64 `json:"bandwidth"`
	RAM      int     `json:"ram"`
}

// ServerWallet represents a server's currency balances.
type ServerWallet struct {
	Crypto float64 `json:"crypto"`
	Data   float64 `json:"data"`
}

// Vulnerability represents a service vulnerability type and level.
type Vulnerability struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

// Service represents a network service running on a server.
type Service struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Port          int            `json:"port"`
	Vulnerable    bool           `json:"vulnerable"`
	Level         int            `json:"level"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

// Role represents a server role or user account on a server.
type Role struct {
	Role  string `json:"role"`
	Level int    `json:"level"`
}

// Server represents a game server that can be scanned, exploited, and accessed.
type Server struct {
	ID            uuid.UUID       `gorm:"type:text;primary_key" json:"id"`
	IP            string          `gorm:"uniqueIndex;not null" json:"ip"`
	LocalIP       string          `gorm:"not null" json:"local_ip"`
	SecurityLevel int             `gorm:"default:100" json:"security_level"`
	Resources     ServerResources `gorm:"type:text;serializer:json" json:"resources"`
	Wallet        ServerWallet    `gorm:"type:text;serializer:json" json:"wallet"`
	Tools         []string        `gorm:"type:text;serializer:json" json:"tools"`
	ConnectedIPs  []string        `gorm:"type:text;serializer:json" json:"connected_ips"`
	Services      []Service       `gorm:"type:text;serializer:json" json:"services"`
	Roles         []Role          `gorm:"type:text;serializer:json" json:"roles"`
	FileSystem    map[string]interface{} `gorm:"type:text;serializer:json" json:"file_system"`
	LocalNetwork  map[string]interface{} `gorm:"type:text;serializer:json" json:"local_network"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// BeforeCreate is a GORM hook that generates a UUID for the server if one doesn't exist.
func (s *Server) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}


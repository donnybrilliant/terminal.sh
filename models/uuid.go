package models

import (
	"github.com/google/uuid"
)

// ParseUUID parses a UUID string
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}


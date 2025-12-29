package models

import (
	"github.com/google/uuid"
)

// ParseUUID parses a UUID string and returns a UUID value.
// Returns an error if the string is not a valid UUID format.
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}


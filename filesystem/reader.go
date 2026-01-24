package filesystem

import (
	"errors"
	"strings"
)

// ErrFileNotFound is returned when a file cannot be found at the specified path.
var ErrFileNotFound = errors.New("file not found")

// ErrNotAFile is returned when the path points to a directory instead of a file.
var ErrNotAFile = errors.New("path is not a file")

// ErrNotADirectory is returned when the path points to a file instead of a directory.
var ErrNotADirectory = errors.New("path is not a directory")

// FileReader reads file content from a filesystem structure.
type FileReader interface {
	// ReadFile reads the content of a file at the given path.
	// Returns ErrFileNotFound if the file doesn't exist.
	ReadFile(path string) (string, error)

	// ListDir lists the entries in a directory at the given path.
	// Returns ErrNotADirectory if the path is not a directory.
	ListDir(path string) ([]string, error)
}

// MapFileReader implements FileReader for map[string]interface{} filesystems.
// This is the format used by server.FileSystem and user.FileSystem in the database.
type MapFileReader struct {
	fs map[string]interface{}
}

// NewMapFileReader creates a new MapFileReader from a filesystem map.
func NewMapFileReader(fs map[string]interface{}) *MapFileReader {
	if fs == nil {
		fs = make(map[string]interface{})
	}
	return &MapFileReader{fs: fs}
}

// ReadFile reads the content of a file at the given path.
func (r *MapFileReader) ReadFile(path string) (string, error) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return "", ErrFileNotFound
	}

	currentLevel := r.fs
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - should be a file
			if file, ok := currentLevel[part].(map[string]interface{}); ok {
				if content, ok := file["content"].(string); ok {
					return content, nil
				}
				// It's a directory, not a file
				return "", ErrNotAFile
			}
			return "", ErrFileNotFound
		}

		// Not the last part - should be a directory
		if dir, ok := currentLevel[part].(map[string]interface{}); ok {
			currentLevel = dir
		} else {
			return "", ErrFileNotFound
		}
	}

	return "", ErrFileNotFound
}

// ListDir lists the entries in a directory at the given path.
func (r *MapFileReader) ListDir(path string) ([]string, error) {
	parts := splitPath(path)

	currentLevel := r.fs

	// Navigate to the target directory
	for _, part := range parts {
		if dir, ok := currentLevel[part].(map[string]interface{}); ok {
			currentLevel = dir
		} else {
			return nil, ErrNotADirectory
		}
	}

	// Collect entries
	entries := make([]string, 0, len(currentLevel))
	for name := range currentLevel {
		entries = append(entries, name)
	}

	return entries, nil
}

// splitPath splits a path into its components, handling leading/trailing slashes.
func splitPath(path string) []string {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return []string{}
	}

	return strings.Split(path, "/")
}

// IsFile checks if the path points to a file (has "content" key).
func (r *MapFileReader) IsFile(path string) bool {
	parts := splitPath(path)
	if len(parts) == 0 {
		return false
	}

	currentLevel := r.fs
	for i, part := range parts {
		if i == len(parts)-1 {
			if file, ok := currentLevel[part].(map[string]interface{}); ok {
				_, hasContent := file["content"].(string)
				return hasContent
			}
			return false
		}

		if dir, ok := currentLevel[part].(map[string]interface{}); ok {
			currentLevel = dir
		} else {
			return false
		}
	}

	return false
}

// IsDir checks if the path points to a directory.
func (r *MapFileReader) IsDir(path string) bool {
	parts := splitPath(path)

	currentLevel := r.fs
	for _, part := range parts {
		if dir, ok := currentLevel[part].(map[string]interface{}); ok {
			currentLevel = dir
		} else {
			return false
		}
	}

	// Check it's not a file (files have "content" key)
	_, hasContent := currentLevel["content"].(string)
	return !hasContent
}

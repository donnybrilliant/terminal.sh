// Package terminal provides terminal UI components and utilities using Bubble Tea.
package terminal

import "strings"

// InputHistory manages command history with up/down navigation for terminal input.
type InputHistory struct {
	commands []string
	index    int
	maxSize  int
}

// NewInputHistory creates a new InputHistory with the specified maximum size.
func NewInputHistory(maxSize int) *InputHistory {
	return &InputHistory{
		commands: make([]string, 0),
		index:    -1,
		maxSize:  maxSize,
	}
}

// Add adds a command to history, skipping duplicates of the last command.
func (h *InputHistory) Add(cmd string) {
	if cmd == "" {
		return
	}
	// Skip if duplicate of last command
	if len(h.commands) > 0 && h.commands[len(h.commands)-1] == cmd {
		return
	}
	h.commands = append(h.commands, cmd)
	// Trim if exceeds max size
	if h.maxSize > 0 && len(h.commands) > h.maxSize {
		h.commands = h.commands[1:]
	}
	h.index = -1
}

// Previous returns the previous command in history (for up arrow navigation).
func (h *InputHistory) Previous() (string, bool) {
	if len(h.commands) == 0 {
		return "", false
	}
	if h.index == -1 {
		h.index = len(h.commands) - 1
	} else if h.index > 0 {
		h.index--
	}
	return h.commands[h.index], true
}

// Next returns the next command in history (for down arrow navigation).
// Returns empty string and false when past the end of history.
func (h *InputHistory) Next() (string, bool) {
	if h.index < 0 {
		return "", false
	}
	h.index++
	if h.index >= len(h.commands) {
		h.index = -1
		return "", false
	}
	return h.commands[h.index], true
}

// Reset resets the history navigation index to the default position.
func (h *InputHistory) Reset() {
	h.index = -1
}

// Autocompleter provides autocomplete functionality (placeholder for future use).
type Autocompleter struct{}

// FindCommonPrefix returns the common prefix of all strings in the slice.
func FindCommonPrefix(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	prefix := items[0]
	for _, item := range items[1:] {
		prefix = commonPrefixOf(prefix, item)
		if prefix == "" {
			break
		}
	}
	return prefix
}

// commonPrefixOf returns the common prefix of two strings
func commonPrefixOf(a, b string) string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:minLen]
}

// FilterByPrefix returns all items that start with the given prefix.
func FilterByPrefix(items []string, prefix string) []string {
	var matches []string
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			matches = append(matches, item)
		}
	}
	return matches
}

// CompleteFromList attempts to autocomplete a prefix from a list of options.
// Returns the completed string and whether completion happened.
func CompleteFromList(prefix string, options []string) (string, bool) {
	matches := FilterByPrefix(options, prefix)

	if len(matches) == 0 {
		return prefix, false
	}

	if len(matches) == 1 {
		// Single match - return it
		return matches[0], true
	}

	// Multiple matches - return common prefix if longer than input
	common := FindCommonPrefix(matches)
	if len(common) > len(prefix) {
		return common, true
	}

	return prefix, false
}

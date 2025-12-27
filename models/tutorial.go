package models

// TutorialStep represents a single step in a tutorial
type TutorialStep struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Commands    []string `json:"commands,omitempty"` // Example commands to run
	Check       string   `json:"check,omitempty"`    // Optional: condition to check if step is complete
}

// Tutorial represents a complete tutorial
type Tutorial struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Steps       []TutorialStep `json:"steps"`
	Prerequisites []string     `json:"prerequisites,omitempty"` // IDs of tutorials that must be completed first
}

// TutorialData represents the complete tutorial data structure
type TutorialData struct {
	Tutorials []Tutorial `json:"tutorials"`
}


package websocket

// Message types for WebSocket communication
const (
	MessageTypeInput   = "input"
	MessageTypeResize  = "resize"
	MessageTypeOutput  = "output"
	MessageTypeClose   = "close"
)

// InputMessage represents keyboard input from the browser
type InputMessage struct {
	Type      string   `json:"type"`
	Key       string   `json:"key,omitempty"`
	Char      string   `json:"char,omitempty"`
	Modifiers []string `json:"modifiers,omitempty"`
}

// ResizeMessage represents terminal size change
type ResizeMessage struct {
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// OutputMessage represents ANSI output to the browser
type OutputMessage struct {
	Type string `json:"type"`
	Data string `json:"data"` // ANSI-encoded string from Bubble Tea View()
}

// CloseMessage represents connection close
type CloseMessage struct {
	Type string `json:"type"`
}


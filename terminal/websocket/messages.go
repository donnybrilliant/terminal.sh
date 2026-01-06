package websocket

// Message types for WebSocket communication between browser and server.
const (
	MessageTypeInput   = "input"
	MessageTypeResize  = "resize"
	MessageTypeOutput  = "output"
	MessageTypeClose   = "close"
	MessageTypeMouse   = "mouse"
)

// InputMessage represents keyboard input from the browser client.
type InputMessage struct {
	Type      string   `json:"type"`
	Key       string   `json:"key,omitempty"`
	Char      string   `json:"char,omitempty"`
	Modifiers []string `json:"modifiers,omitempty"`
}

// ResizeMessage represents a terminal size change from the browser client.
type ResizeMessage struct {
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// OutputMessage represents ANSI-encoded terminal output to send to the browser client.
type OutputMessage struct {
	Type string `json:"type"`
	Data string `json:"data"` // ANSI-encoded string from Bubble Tea View()
}

// CloseMessage represents a connection close request from the browser client.
type CloseMessage struct {
	Type string `json:"type"`
}

// MouseMessage represents mouse events (e.g., scroll wheel) from the browser client.
type MouseMessage struct {
	Type   string `json:"type"`
	Button string `json:"button"` // "wheelUp", "wheelDown"
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

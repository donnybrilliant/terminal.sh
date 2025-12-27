package websocket

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"terminal-sh/config"
	"terminal-sh/database"
	"terminal-sh/services"

	"github.com/charmbracelet/lipgloss"
)

var (
	successLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
	infoLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)

// StartHTTPServer starts the HTTP server for serving static files and WebSocket connections
func StartHTTPServer(cfg *config.Config, db *database.Database, chatService *services.ChatService) error {
	userService := services.NewUserService(db, cfg.JWTSecret)

	// Determine web directory path (relative to working directory)
	// Try multiple possible locations
	webDir := "web"
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		// Try relative to executable
		execPath, _ := os.Executable()
		execDir := filepath.Dir(execPath)
		webDir = filepath.Join(execDir, "web")
		if _, err := os.Stat(webDir); os.IsNotExist(err) {
			// Fallback: try current directory
			webDir = "./web"
		}
	}

	// Serve static files
	fileServer := http.FileServer(http.Dir(webDir))
	http.Handle("/", fileServer)

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if err := HandleWebSocket(w, r, db, userService, chatService); err != nil {
			log.Printf("WebSocket error: %v", err)
		}
	})

	addr := cfg.WebHost + ":" + fmt.Sprintf("%d", cfg.WebPort)
	fmt.Println(infoLogStyle.Render(fmt.Sprintf("HTTP/WebSocket server listening on %s", addr)))
	fmt.Println(successLogStyle.Render("✓") + " " + infoLogStyle.Render(fmt.Sprintf("WebSocket endpoint: ws://%s/ws", addr)))
	fmt.Println(successLogStyle.Render("✓") + " Static files served from /")
	
	return http.ListenAndServe(addr, nil)
}


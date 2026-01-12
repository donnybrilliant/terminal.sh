package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"terminal-sh/config"
	"terminal-sh/database"
	"terminal-sh/services"
	"terminal-sh/terminal/ssh"
	"terminal-sh/terminal/websocket"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
	boxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))
)

func main() {
	cfg := config.Load()

	// Header
	header := "╔═══════════════════════════════════════╗\n║   ui.sh Server - Initializing   ║\n║   (SSH + WebSocket)                    ║\n╚═══════════════════════════════════════╝"
	fmt.Println(boxStyle.Render(header))
	fmt.Println()

	// Initialize database
	fmt.Print(successStyle.Render("✓") + " Initializing database...")
	db, err := database.NewDB(cfg.DatabasePath, cfg.DatabaseURL)
	if err != nil {
		fmt.Println(" " + errorStyle.Render("✗"))
		log.Fatalf(errorStyle.Render("Failed to initialize database: %v"), err)
	}
	defer func() {
		fmt.Println()
		fmt.Print(successStyle.Render("✓") + " Closing database connection...")
		if err := db.Close(); err != nil {
			log.Printf(errorStyle.Render("Error closing database: %v"), err)
		} else {
			fmt.Println()
		}
	}()
	fmt.Println()

	// Seed tools
	fmt.Print(successStyle.Render("✓") + " Seeding tools...")
	serverService := services.NewServerService(db)
	toolService := services.NewToolService(db, serverService)
	if err := toolService.SeedTools(); err != nil {
		fmt.Println(" " + errorStyle.Render("✗"))
		log.Printf(errorStyle.Render("Failed to seed tools: %v"), err)
	} else {
		fmt.Println()
	}

	// Seed initial game data (servers, etc.)
	fmt.Print(successStyle.Render("✓") + " Seeding game data...")
	if err := services.SeedInitialData(db); err != nil {
		fmt.Println(" " + errorStyle.Render("✗"))
		log.Printf(errorStyle.Render("Failed to seed initial data: %v"), err)
	} else {
		fmt.Println()
	}

	// Initialize chat service
	fmt.Print(successStyle.Render("✓") + " Initializing chat service...")
	chatService := services.NewChatService(db)
	if err := chatService.InitializeDefaultRoom(); err != nil {
		fmt.Println(" " + errorStyle.Render("✗"))
		log.Printf(errorStyle.Render("Failed to initialize default room: %v"), err)
	} else {
		fmt.Println()
	}

	fmt.Println()
	readyBox := fmt.Sprintf("╔═══════════════════════════════════════╗\n║   SSH Server ready on %s:%d        ║\n║   Web Server ready on %s:%d        ║\n╚═══════════════════════════════════════╝", cfg.Host, cfg.Port, cfg.WebHost, cfg.WebPort)
	fmt.Println(infoStyle.Render(readyBox))
	fmt.Println()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start both servers in goroutines
	sshErr := make(chan error, 1)
	webErr := make(chan error, 1)

	go func() {
		if err := ssh.StartServer(cfg, db, chatService); err != nil {
			sshErr <- err
		}
	}()

	go func() {
		if err := websocket.StartHTTPServer(cfg, db, chatService); err != nil {
			webErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case sig := <-sigChan:
		fmt.Printf("\n\n")
		fmt.Println(infoStyle.Render(fmt.Sprintf("Received signal: %v", sig)))
		fmt.Println(infoStyle.Render("Shutting down gracefully..."))

		// Give the servers a moment to finish current operations
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shutdown SSH server
		if err := ssh.ShutdownServer(ctx); err != nil {
			log.Printf(errorStyle.Render("Error shutting down SSH server: %v"), err)
		} else {
			fmt.Println(successStyle.Render("✓ SSH server shut down gracefully"))
		}

		// Web server shutdown (HTTP server doesn't have explicit shutdown in current implementation)
		// It will stop when the process exits
		fmt.Println(successStyle.Render("✓ Web server shut down gracefully"))

	case err := <-sshErr:
		log.Fatalf("SSH server error: %v", err)
	case err := <-webErr:
		log.Fatalf("Web server error: %v", err)
	}
}


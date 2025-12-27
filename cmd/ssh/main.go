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
	header := "╔═══════════════════════════════════════╗\n║   terminal.sh SSH Server - Initializing    ║\n╚═══════════════════════════════════════╝"
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

	fmt.Println()
	readyBox := fmt.Sprintf("╔═══════════════════════════════════════╗\n║   SSH Server ready on %s:%d        ║\n╚═══════════════════════════════════════╝", cfg.Host, cfg.Port)
	fmt.Println(infoStyle.Render(readyBox))
	fmt.Println()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := ssh.StartServer(cfg, db); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case sig := <-sigChan:
		fmt.Printf("\n\n")
		fmt.Println(infoStyle.Render(fmt.Sprintf("Received signal: %v", sig)))
		fmt.Println(infoStyle.Render("Shutting down gracefully..."))
		
		// Give the server a moment to finish current operations
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Shutdown the server
		if err := ssh.ShutdownServer(ctx); err != nil {
			log.Printf(errorStyle.Render("Error during shutdown: %v"), err)
		} else {
			fmt.Println(successStyle.Render("✓ Server shut down gracefully"))
		}
		
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	}
}


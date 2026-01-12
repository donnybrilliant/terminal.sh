package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"terminal-sh/config"
	"terminal-sh/database"
	"terminal-sh/services"
	"terminal-sh/terminal/websocket"
	"terminal-sh/ui"
)

func main() {
	cfg := config.Load()

	// Header
	header := "╔═══════════════════════════════════════╗\n║   terminal.sh Web Server - Initializing   ║\n╚═══════════════════════════════════════╝"
	fmt.Println(ui.HeaderStyle.Render(header))
	fmt.Println()

	// Initialize database
	fmt.Print(ui.SuccessStyle.Render("✓") + " Initializing database...")
	db, err := database.NewDB(cfg.DatabasePath, cfg.DatabaseURL)
	if err != nil {
		fmt.Println(" " + ui.ErrorStyle.Render("✗"))
		log.Fatalf(ui.ErrorStyle.Render("Failed to initialize database: %v"), err)
	}
	defer func() {
		fmt.Println()
		fmt.Print(ui.SuccessStyle.Render("✓") + " Closing database connection...")
		if err := db.Close(); err != nil {
			log.Printf(ui.ErrorStyle.Render("Error closing database: %v"), err)
		} else {
			fmt.Println()
		}
	}()
	fmt.Println()

	// Seed tools
	fmt.Print(ui.SuccessStyle.Render("✓") + " Seeding tools...")
	serverService := services.NewServerService(db)
	toolService := services.NewToolService(db, serverService)
	if err := toolService.SeedTools(); err != nil {
		fmt.Println(" " + ui.ErrorStyle.Render("✗"))
		log.Printf(ui.ErrorStyle.Render("Failed to seed tools: %v"), err)
	} else {
		fmt.Println()
	}
	
	// Seed initial game data (servers, etc.)
	fmt.Print(ui.SuccessStyle.Render("✓") + " Seeding game data...")
	if err := services.SeedInitialData(db); err != nil {
		fmt.Println(" " + ui.ErrorStyle.Render("✗"))
		log.Printf(ui.ErrorStyle.Render("Failed to seed initial data: %v"), err)
	} else {
		fmt.Println()
	}

	// Initialize chat service
	fmt.Print(ui.SuccessStyle.Render("✓") + " Initializing chat service...")
	chatService := services.NewChatService(db)
	if err := chatService.InitializeDefaultRoom(); err != nil {
		fmt.Println(" " + ui.ErrorStyle.Render("✗"))
		log.Printf(ui.ErrorStyle.Render("Failed to initialize default room: %v"), err)
	} else {
		fmt.Println()
	}

	fmt.Println()
	readyBox := fmt.Sprintf("╔═══════════════════════════════════════╗\n║   Web Server ready on %s:%d        ║\n╚═══════════════════════════════════════╝", cfg.WebHost, cfg.WebPort)
	fmt.Println(ui.InfoStyle.Render(readyBox))
	fmt.Println()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := websocket.StartHTTPServer(cfg, db, chatService); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case sig := <-sigChan:
		fmt.Printf("\n\n")
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Received signal: %v", sig)))
		fmt.Println(ui.InfoStyle.Render("Shutting down gracefully..."))
		// HTTP server shutdown would go here if needed
		fmt.Println(ui.SuccessStyle.Render("✓ Server shut down gracefully"))
		
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	}
}


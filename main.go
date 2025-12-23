package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ssh4xx-go/config"
	"ssh4xx-go/database"
	"ssh4xx-go/services"
	"ssh4xx-go/terminal"
)

func main() {
	cfg := config.Load()

	fmt.Println("╔═══════════════════════════════════════╗")
	fmt.Println("║   SSH4XX Server - Initializing        ║")
	fmt.Println("╚═══════════════════════════════════════╝")
	fmt.Println()

	// Initialize database
	fmt.Print("Initializing database... ")
	if err := database.Init(cfg.DatabasePath); err != nil {
		log.Fatalf("\n✗ Failed to initialize database: %v", err)
	}
	defer func() {
		fmt.Println("\nClosing database connection...")
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		} else {
			fmt.Println("✓ Database closed")
		}
	}()
	fmt.Println("✓")

	// Seed tools
	fmt.Print("Seeding tools... ")
	serverService := services.NewServerService()
	toolService := services.NewToolService(serverService)
	if err := toolService.SeedTools(); err != nil {
		log.Printf("\n✗ Failed to seed tools: %v", err)
	} else {
		fmt.Println("✓")
	}
	
	// Seed initial game data (servers, etc.)
	fmt.Print("Seeding game data... ")
	if err := services.SeedInitialData(); err != nil {
		log.Printf("\n✗ Failed to seed initial data: %v", err)
	} else {
		fmt.Println("✓")
	}

	fmt.Println()
	fmt.Printf("╔═══════════════════════════════════════╗\n")
	fmt.Printf("║   Server ready on %s:%d        ║\n", cfg.Host, cfg.Port)
	fmt.Printf("╚═══════════════════════════════════════╝\n")
	fmt.Println()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := terminal.StartServer(cfg); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case sig := <-sigChan:
		fmt.Printf("\n\nReceived signal: %v\n", sig)
		fmt.Println("Shutting down gracefully...")
		
		// Give the server a moment to finish current operations
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Shutdown the server
		if err := terminal.ShutdownServer(ctx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		} else {
			fmt.Println("✓ Server shut down gracefully")
		}
		
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	}
}


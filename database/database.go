package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"terminal-sh/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database wraps gorm.DB for dependency injection
type Database struct {
	*gorm.DB
}

// NewDB creates a new database connection and runs migrations
// If dbURL is provided and starts with "postgres://" or "postgresql://", it uses PostgreSQL
// Otherwise, it uses SQLite with dbPath
func NewDB(dbPath, dbURL string) (*Database, error) {
	var gormDB *gorm.DB
	var err error

	// Use PostgreSQL if DATABASE_URL is provided
	if dbURL != "" && (strings.HasPrefix(dbURL, "postgres://") || strings.HasPrefix(dbURL, "postgresql://")) {
		gormDB, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent), // Silent mode - only log errors
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
		}
	} else {
		// Use SQLite
		// Create directory if it doesn't exist (skip for in-memory databases)
		if dbPath != ":memory:" {
			dir := filepath.Dir(dbPath)
			if dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return nil, fmt.Errorf("failed to create database directory: %w", err)
				}
			}
		}

		gormDB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent), // Silent mode - only log errors
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
		}
	}

	db := &Database{DB: gormDB}

	// Run migrations
	if err := db.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Migrate runs database migrations
func (db *Database) Migrate() error {
	// Auto-migrate all models
	err := db.AutoMigrate(
		&models.User{},
		&models.Tool{},
		&models.UserTool{},
		&models.UserToolState{},
		&models.Shop{},
		&models.ShopItem{},
		&models.UserPurchase{},
		&models.Patch{},
		&models.UserPatch{},
		&models.Server{},
		&models.UserAchievement{},
		&models.ExploitedServer{},
		&models.ActiveMiner{},
		&models.Session{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}
	
	return nil
}

// Close closes the database connection
func (db *Database) Close() error {
	if db.DB != nil {
		sqlDB, err := db.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}


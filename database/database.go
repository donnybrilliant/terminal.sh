package database

import (
	"fmt"

	"ssh4xx-go/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init initializes the database connection and runs migrations
func Init(dbPath string) error {
	var err error
	
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silent mode - only log errors
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Migrate runs database migrations
func Migrate() error {
	// Auto-migrate all models
	err := DB.AutoMigrate(
		&models.User{},
		&models.Tool{},
		&models.UserTool{},
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
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}


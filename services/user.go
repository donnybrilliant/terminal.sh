package services

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"terminal-sh/auth"
	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserService handles user-related operations
type UserService struct {
	db           *database.Database
	tokenManager *auth.TokenManager
}

// NewUserService creates a new user service
func NewUserService(db *database.Database, jwtSecret string) *UserService {
	return &UserService{
		db:           db,
		tokenManager: auth.NewTokenManager(jwtSecret),
	}
}

// Register creates a new user
func (s *UserService) Register(username, password string) (*models.User, error) {
	// Validate username
	if username == "" || username == "guest" {
		return nil, fmt.Errorf("invalid username")
	}

	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Hash password
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate IP addresses and MAC
	ip := generateIP()
	localIP := generateLocalIP()
	mac := generateMAC()

	// Create user
	user := &models.User{
		Username:     username,
		PasswordHash: passwordHash,
		IP:           ip,
		LocalIP:      localIP,
		MAC:          mac,
		Level:        0,
		Experience:   0,
		Resources: models.Resources{
			CPU:      200,
			Bandwidth: 300,
			RAM:      24,
		},
		Wallet: models.Wallet{
			Crypto: 15,
			Data:   1200,
		},
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns a JWT token
func (s *UserService) Login(username, password string) (*models.User, string, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Auto-register on first login attempt
			user, err := s.Register(username, password)
			if err != nil {
				return nil, "", fmt.Errorf("registration failed: %w", err)
			}
			token, err := s.tokenManager.GenerateToken(user.ID, user.Username)
			if err != nil {
				return nil, "", fmt.Errorf("failed to generate token: %w", err)
			}
			return user, token, nil
		}
		return nil, "", fmt.Errorf("failed to find user: %w", err)
	}

	// Check password
	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("invalid password")
	}

	// Generate token
	token, err := s.tokenManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &user, token, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("Tools").Preload("Achievements").First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("Tools").Preload("Achievements").First(&user, "username = ?", username).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUsername updates a user's username
func (s *UserService) UpdateUsername(userID uuid.UUID, newUsername string) error {
	if newUsername == "" || newUsername == "guest" {
		return fmt.Errorf("invalid username")
	}

	// Check if username is already taken
	var existingUser models.User
	if err := s.db.Where("username = ? AND id != ?", newUsername, userID).First(&existingUser).Error; err == nil {
		return fmt.Errorf("username already taken")
	}

	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("username", newUsername).Error
}

// AddExperience adds experience to a user and levels them up if needed
func (s *UserService) AddExperience(userID uuid.UUID, amount int) error {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	user.Experience += amount
	
	// Level up calculation (simple: 100 exp per level)
	newLevel := user.Experience / 100
	if newLevel > user.Level {
		user.Level = newLevel
	}

	return s.db.Model(user).Updates(map[string]interface{}{
		"experience": user.Experience,
		"level":      user.Level,
	}).Error
}

// Helper functions for generating IPs and MAC addresses

func generateIP() string {
	// Generate a random IP in the range 1.0.0.0 - 255.255.255.255
	// For simplicity, we'll use a deterministic approach based on random numbers
	part1 := randomInt(1, 255)
	part2 := randomInt(0, 255)
	part3 := randomInt(0, 255)
	part4 := randomInt(1, 255)
	return fmt.Sprintf("%d.%d.%d.%d", part1, part2, part3, part4)
}

func generateLocalIP() string {
	// Generate a local IP in the 10.0.0.0/8 range
	part2 := randomInt(0, 255)
	part3 := randomInt(0, 255)
	part4 := randomInt(1, 254)
	return fmt.Sprintf("10.%d.%d.%d", part2, part3, part4)
}

func generateMAC() string {
	// Generate a random MAC address
	mac := make([]byte, 6)
	rand.Read(mac)
	mac[0] = (mac[0] | 2) & 0xfe // Set locally administered bit
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

func randomInt(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return int(n.Int64()) + min
}

// ValidateToken validates a JWT token and returns the user
func (s *UserService) ValidateToken(tokenString string) (*models.User, error) {
	claims, err := s.tokenManager.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return s.GetUserByID(claims.UserID)
}


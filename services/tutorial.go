package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"terminal-sh/models"
)

// TutorialService handles tutorial-related operations including loading and retrieval.
type TutorialService struct {
	tutorials []models.Tutorial
	dataPath  string
}

// NewTutorialService creates a new TutorialService and loads tutorials from the specified path.
func NewTutorialService(dataPath string) (*TutorialService, error) {
	service := &TutorialService{
		dataPath: dataPath,
	}
	
	// Load tutorials from file
	if err := service.LoadTutorials(); err != nil {
		return nil, fmt.Errorf("failed to load tutorials: %w", err)
	}
	
	return service, nil
}

// LoadTutorials loads tutorials from the JSON file
func (s *TutorialService) LoadTutorials() error {
	// Default to data/seed/tutorials.json if not specified
	path := s.dataPath
	if path == "" {
		path = "data/seed/tutorials.json"
	}
	
	// Check if file exists, if not create default
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := s.createDefaultTutorials(path); err != nil {
			return fmt.Errorf("failed to create default tutorials: %w", err)
		}
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tutorial file: %w", err)
	}
	
	var tutorialData models.TutorialData
	if err := json.Unmarshal(data, &tutorialData); err != nil {
		return fmt.Errorf("failed to parse tutorial file: %w", err)
	}
	
	s.tutorials = tutorialData.Tutorials
	return nil
}

// ReloadTutorials reloads tutorials from the file
func (s *TutorialService) ReloadTutorials() error {
	return s.LoadTutorials()
}

// GetAllTutorials returns all available tutorials
func (s *TutorialService) GetAllTutorials() []models.Tutorial {
	return s.tutorials
}

// GetTutorialByID returns a tutorial by its ID
func (s *TutorialService) GetTutorialByID(id string) (*models.Tutorial, error) {
	for _, tutorial := range s.tutorials {
		if tutorial.ID == id {
			return &tutorial, nil
		}
	}
	return nil, fmt.Errorf("tutorial not found: %s", id)
}

// GetTutorialPath returns the path to the tutorial file
func (s *TutorialService) GetTutorialPath() string {
	if s.dataPath != "" {
		return s.dataPath
	}
	return "data/seed/tutorials.json"
}

// createDefaultTutorials creates a default tutorial file
func (s *TutorialService) createDefaultTutorials(path string) error {
	defaultTutorials := models.TutorialData{
		Tutorials: []models.Tutorial{
			{
				ID:          "getting_started",
				Name:        "Getting Started",
				Description: "Learn the basics of terminal.sh",
				Steps: []models.TutorialStep{
					{
						ID:          1,
						Title:       "Welcome to terminal.sh",
						Description: "terminal.sh is a hacking simulation game. You'll learn to scan networks, exploit vulnerabilities, and mine cryptocurrency.",
					},
					{
						ID:          2,
						Title:       "Basic Commands",
						Description: "Try these basic commands:\n  - `help` - Show all available commands\n  - `userinfo` - View your user information\n  - `ifconfig` - View your network configuration\n  - `wallet` - Check your wallet balance",
						Commands: []string{"help", "userinfo", "ifconfig", "wallet"},
					},
					{
						ID:          3,
						Title:       "Scanning the Network",
						Description: "Use `scan` to discover servers on the internet. This is your first step to finding targets.",
						Commands: []string{"scan"},
					},
					{
						ID:          4,
						Title:       "Scanning a Server",
						Description: "Use `scan <targetIP>` to scan a specific server for services and vulnerabilities. Try scanning one of the servers you found.",
						Commands: []string{"scan 1.1.1.1"},
					},
				},
			},
			{
				ID:          "exploitation",
				Name:        "Exploitation Basics",
				Description: "Learn how to exploit servers",
				Prerequisites: []string{"getting_started"},
				Steps: []models.TutorialStep{
					{
						ID:          1,
						Title:       "Getting Tools",
						Description: "Tools are essential for exploitation. The 'repo' server contains all available tools. Use `get repo <toolName>` to download tools.",
						Commands: []string{"get repo password_cracker"},
					},
					{
						ID:          2,
						Title:       "Listing Your Tools",
						Description: "Use `tools` to see all tools you own. You need to own a tool before you can use it.",
						Commands: []string{"tools"},
					},
					{
						ID:          3,
						Title:       "Exploiting a Server",
						Description: "Once you have a tool, you can exploit servers. First, scan a server to see its vulnerabilities, then use the appropriate tool. Example: `password_cracker <targetIP>`",
						Commands: []string{"password_cracker 1.1.1.1"},
					},
					{
						ID:          4,
						Title:       "SSH Access",
						Description: "After successfully exploiting a server, you can SSH into it using `ssh <targetIP>`. This gives you access to the server's filesystem.",
						Commands: []string{"ssh 1.1.1.1"},
					},
				},
			},
			{
				ID:          "mining",
				Name:        "Cryptocurrency Mining",
				Description: "Learn how to mine cryptocurrency for passive income",
				Prerequisites: []string{"exploitation"},
				Steps: []models.TutorialStep{
					{
						ID:          1,
						Title:       "Getting the Crypto Miner",
						Description: "Download the crypto_miner tool from the repo server.",
						Commands: []string{"get repo crypto_miner"},
					},
					{
						ID:          2,
						Title:       "Starting a Miner",
						Description: "Use `crypto_miner <targetIP>` to start mining on an exploited server. The server must have enough resources (CPU, RAM, bandwidth).",
						Commands: []string{"crypto_miner 1.1.1.1"},
					},
					{
						ID:          3,
						Title:       "Checking Active Miners",
						Description: "Use `miners` to see all your active mining sessions. Miners generate cryptocurrency over time.",
						Commands: []string{"miners"},
					},
					{
						ID:          4,
						Title:       "Stopping a Miner",
						Description: "Use `stop_mining <targetIP>` to stop a miner when needed.",
						Commands: []string{"stop_mining 1.1.1.1"},
					},
				},
			},
			{
				ID:          "advanced_tools",
				Name:        "Advanced Tools",
				Description: "Learn about advanced exploitation tools",
				Prerequisites: []string{"exploitation"},
				Steps: []models.TutorialStep{
					{
						ID:          1,
						Title:       "Information Gathering",
						Description: "Tools like `user_enum` and `lan_sniffer` help you gather information about servers before exploitation.",
						Commands: []string{"user_enum 1.1.1.1", "lan_sniffer 1.1.1.1"},
					},
					{
						ID:          2,
						Title:       "Multi-Vulnerability Exploitation",
						Description: "The `exploit_kit` and `advanced_exploit_kit` tools can exploit multiple vulnerability types at once.",
						Commands: []string{"get repo exploit_kit", "exploit_kit 1.1.1.1"},
					},
					{
						ID:          3,
						Title:       "Web Exploitation",
						Description: "Tools like `sql_injector` and `xss_exploit` target HTTP services specifically.",
						Commands: []string{"get repo sql_injector", "sql_injector 1.1.1.1"},
					},
					{
						ID:          4,
						Title:       "Network Analysis",
						Description: "Use `packet_capture` and `packet_decoder` to analyze network traffic.",
						Commands: []string{"get repo packet_capture", "packet_capture 1.1.1.1"},
					},
				},
			},
		},
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	
	data, err := json.MarshalIndent(defaultTutorials, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tutorials: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write tutorial file: %w", err)
	}
	
	s.tutorials = defaultTutorials.Tutorials
	return nil
}


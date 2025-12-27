.PHONY: build build-ssh build-web build-all dev run run-ssh run-web run-all clean

# Build directory
BIN_DIR := bin
BINARY_SSH := $(BIN_DIR)/terminal.sh-ssh
BINARY_WEB := $(BIN_DIR)/terminal.sh-web
BINARY_ALL := $(BIN_DIR)/terminal.sh

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# ANSI color codes
GREEN := \033[32;1m
CYAN := \033[36m
MAGENTA := \033[35;1m
RESET := \033[0m

build-ssh: ## Build SSH server only
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)Building SSH server...$(RESET)"
	@go build -o $(BINARY_SSH) ./cmd/ssh
	@echo "$(GREEN)✓$(RESET) $(CYAN)SSH server built: $(BINARY_SSH)$(RESET)"

build-web: ## Build Web server only
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)Building Web server...$(RESET)"
	@go build -o $(BINARY_WEB) ./cmd/web
	@echo "$(GREEN)✓$(RESET) $(CYAN)Web server built: $(BINARY_WEB)$(RESET)"

build-all: ## Build all binaries (SSH, Web, Combined)
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)Building all binaries...$(RESET)"
	@go build -o $(BINARY_SSH) ./cmd/ssh
	@go build -o $(BINARY_WEB) ./cmd/web
	@go build -o $(BINARY_ALL) ./cmd/all
	@echo "$(GREEN)✓$(RESET) $(CYAN)All binaries built in $(BIN_DIR)/$(RESET)"

build: build-all ## Alias for build-all

run-ssh: build-ssh ## Build and run SSH server only
	@echo ""
	@echo "$(CYAN)Starting SSH server...$(RESET)"
	@./$(BINARY_SSH)

run-web: build-web ## Build and run Web server only
	@echo ""
	@echo "$(CYAN)Starting Web server...$(RESET)"
	@./$(BINARY_WEB)

run-all: build-all ## Build and run combined server (SSH + Web)
	@echo ""
	@echo "$(CYAN)Starting combined server (SSH + Web)...$(RESET)"
	@./$(BINARY_ALL)

dev: build-all ## Build all and start combined server (development mode)
	@echo ""
	@echo "$(MAGENTA)╔═══════════════════════════════════════╗$(RESET)"
	@echo "$(MAGENTA)║   Development Mode - Starting Server  ║$(RESET)"
	@echo "$(MAGENTA)╚═══════════════════════════════════════╝$(RESET)"
	@echo ""
	@./$(BINARY_ALL)

run: run-all ## Alias for run-all

clean: ## Remove all built binaries
	@echo "$(CYAN)Cleaning binaries...$(RESET)"
	@rm -f $(BINARY_SSH) $(BINARY_WEB) $(BINARY_ALL)
	@echo "$(GREEN)✓$(RESET) $(CYAN)Cleaned$(RESET)"


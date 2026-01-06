# Implementation Summary

## Seed Data Refactoring (Consistent JSON-Based Seeding)

All game seed data has been refactored to use JSON files for consistency and maintainability:

### Structure

All seed data is now stored in `data/seed/` directory:
- `data/seed/tools.json` - Tool definitions
- `data/seed/servers.json` - Initial servers (repo, test)
- `data/seed/shops.json` - Shop definitions and items
- `data/seed/patches.json` - Patch/upgrade definitions
- `data/seed/tutorials.json` - Tutorial content

### Benefits

- **Consistent**: All seed data uses the same JSON-based pattern
- **Maintainable**: Easy to edit JSON files without code changes
- **Production-friendly**: Update JSON files and restart server to apply changes
- **Version controlled**: All seed data is tracked in git
- **Database-backed**: Data is seeded to database on startup (idempotent)

### Seeding Process

All seed functions:
1. Load data from JSON files in `data/seed/`
2. Parse JSON into Go models
3. Seed to database (idempotent - checks if exists, skips if already present)

### Editing Seed Data

To modify game data:
1. Edit the appropriate JSON file in `data/seed/`
2. Restart the server
3. Seeding logic will update the database

## Architecture Refactoring

### Dual Server Architecture

The project has been refactored to support separate SSH and WebSocket servers that can be deployed independently:

#### Key Changes:

1. **Database Refactoring**
   - Removed global database variable
   - Implemented dependency injection with `database.Database` struct
   - All services now accept `*database.Database` in constructors
   - Enables separate deployment with shared or separate databases

2. **Separate Binaries**
   - `cmd/ssh/main.go` - SSH server entry point
   - `cmd/web/main.go` - WebSocket/HTTP server entry point
   - Both can be built and deployed independently

3. **Terminal Code Organization**
   - `terminal/ssh/` - SSH-specific server code
   - `terminal/websocket/` - WebSocket bridge and HTTP server
   - `terminal/` - Shared Bubble Tea models (login, shell)

4. **WebSocket Implementation**
   - Full Bubble Tea over WebSocket
   - Browser-based terminal using xterm.js
   - Identical functionality to SSH interface
   - Real-time bidirectional communication

#### Deployment Options:

- **Monolithic**: Run both servers on same machine/container
- **Separate Containers**: Deploy SSH and Web servers in different containers
- **Shared Database**: Both servers can use same SQLite file or PostgreSQL instance
- **Separate Databases**: Each server can use its own database

#### Configuration:

- SSH Server: `HOST`, `PORT`, `HOSTKEY_PATH`, `DATABASE_PATH`, `JWT_SECRET`
- Web Server: `WEB_HOST`, `WEB_PORT`, `DATABASE_PATH`, `DATABASE_URL`, `JWT_SECRET`

## Completed Improvements

### 1. Missing Tools Implementation

Added the following missing tools that were referenced in `ORIGINAL_PROJECT_REFERENCE.md` but not implemented:

#### Tools Added:

- **password_sniffer** - Sniff and crack passwords from user roles
- **advanced_exploit_kit** - Advanced multi-vulnerability exploitation
- **sql_injector** - Perform SQL injection attacks on HTTP services
- **xss_exploit** - Exploit XSS vulnerabilities on HTTP services
- **packet_capture** - Capture network packets
- **packet_decoder** - Decode captured packets

#### Implementation Details:

- All tools added to database seed (`services/tool.go`)
- Command handlers implemented (`cmd/tool_commands.go`)
- Commands registered in command router (`cmd/commands.go`)

### 2. Tutorial System

Created a comprehensive tutorial system that allows you to:

- Edit tutorials without code changes
- Display tutorials to users
- Guide players through the game

#### Files Created:

- `models/tutorial.go` - Tutorial data models
- `services/tutorial.go` - Tutorial service for loading and managing tutorials
- `tutorials.json` - JSON file containing tutorial data (auto-generated on first run)

#### Features:

- **Editable Tutorials**: Tutorials are stored in `tutorials.json` which can be edited directly
- **Dynamic Loading**: Tutorials are reloaded each time the `tutorial` command is run
- **Structured Format**: Each tutorial has:
  - ID, name, and description
  - Multiple steps with titles and descriptions
  - Example commands for each step
  - Prerequisites (tutorials that must be completed first)

#### Usage:

```bash
# List all available tutorials
tutorial

# View a specific tutorial
tutorial getting_started
tutorial exploitation
tutorial mining
tutorial advanced_tools
```

#### Default Tutorials Included:

1. **getting_started** - Basic introduction and commands
2. **exploitation** - How to exploit servers (requires getting_started)
3. **mining** - Cryptocurrency mining guide (requires exploitation)
4. **advanced_tools** - Advanced tool usage (requires exploitation)

## How to Edit Tutorials

1. **Locate the tutorial file**: `tutorials.json` in the project root
2. **Edit the JSON file**: Use any text editor to modify tutorials
3. **Reload**: The `tutorial` command automatically reloads the file each time it's run

### Tutorial File Structure:

```json
{
  "tutorials": [
    {
      "id": "tutorial_id",
      "name": "Tutorial Name",
      "description": "Description of what this tutorial teaches",
      "prerequisites": ["other_tutorial_id"],
      "steps": [
        {
          "id": 1,
          "title": "Step Title",
          "description": "Detailed explanation",
          "commands": ["example command", "another command"]
        }
      ]
    }
  ]
}
```

### Example: Adding a New Tutorial

Add a new tutorial object to the `tutorials` array in `tutorials.json`:

```json
{
  "id": "my_new_tutorial",
  "name": "My New Tutorial",
  "description": "This tutorial teaches something new",
  "steps": [
    {
      "id": 1,
      "title": "First Step",
      "description": "Learn about X",
      "commands": ["command1", "command2"]
    }
  ]
}
```

Then users can access it with: `tutorial my_new_tutorial`

## Command Updates

### New Commands Available:

- `tutorial` - List all tutorials
- `tutorial <id>` - View a specific tutorial
- `password_sniffer <targetIP>` - Sniff passwords from roles
- `advanced_exploit_kit <targetIP>` - Advanced exploitation
- `sql_injector <targetIP>` - SQL injection attacks
- `xss_exploit <targetIP>` - XSS exploitation
- `packet_capture <targetIP>` - Capture packets
- `packet_decoder <targetIP>` - Decode packets

### Updated Commands:

- `help` - Now includes tutorial command information

## Testing

To test the implementation:

1. **Build the servers**:

   ```bash
   # Build SSH server
   go build -o bin/terminal.sh-ssh ./cmd/ssh
   
   # Build Web server
   go build -o bin/terminal.sh-web ./cmd/web
   ```

2. **Run the servers**:

   ```bash
   # Terminal 1 - SSH server
   ./bin/terminal.sh-ssh
   
   # Terminal 2 - Web server
   ./bin/terminal.sh-web
   ```

3. **Connect via SSH**:

   ```bash
   ssh -p 2222 username@localhost
   ```

4. **Connect via Web**:

   Open `http://localhost:8080` in your browser

4. **Test tutorials**:

   ```bash
   tutorial
   tutorial getting_started
   ```

5. **Test new tools** (after downloading them):
   ```bash
   get repo password_sniffer
   password_sniffer 1.1.1.1
   ```

## Notes

- The tutorial system automatically creates `tutorials.json` with default tutorials on first run
- Tutorials are reloaded from the file each time the `tutorial` command is executed
- All new tools follow the same pattern as existing tools
- Tool commands require the user to own the tool before use
- Experience points are awarded for using tools (varies by tool)

## Filesystem Implementation

The game features a sophisticated virtual filesystem (VFS) system that supports both user-local and server-shared filesystems with efficient storage and protection mechanisms.

### Architecture Overview

The filesystem implementation uses a **standard base + merge** approach:

1. **Standard Filesystem**: Every VFS starts with a standard structure (system directories, commands, etc.)
2. **Merge on Load**: User/server-specific changes are merged onto the standard base
3. **Save Only Changes**: Only non-standard files/directories are persisted to the database
4. **Deletion Protection**: Standard filesystem nodes cannot be deleted

### Key Components

#### 1. Standard Filesystem Structure

Every VFS includes:
- `/bin` - System commands (ls, cd, pwd, etc.)
- `/usr/bin` - User-acquired tools (downloaded via `get` command)
- `/home/<username>` - User home directory
- `/home/<username>/README.txt` - Welcome file

Standard paths are tracked in a `standardPaths` map and cannot be deleted.

#### 2. User Filesystem

Each user has a private filesystem stored in `User.FileSystem` (JSON field in database):

- **Location**: `models.User.FileSystem` (map[string]interface{})
- **Loading**: On login, user's saved filesystem is merged with standard structure
- **Persistence**: Only user-created files are saved (not standard structure)
- **Privacy**: Each user's filesystem is completely private

#### 3. Server Filesystem

Each server has a shared filesystem stored in `Server.FileSystem`:

- **Location**: `models.Server.FileSystem` (map[string]interface{})
- **Loading**: When SSHing into a server, its filesystem is loaded and merged
- **Persistence**: Only server-specific files are saved
- **Global Access**: All players who SSH into the same server see the same filesystem

### Implementation Details

#### VFS Structure

```go
type VFS struct {
    Root           *Node
    Current        *Node
    username       string
    standardPaths  map[string]bool  // Tracks standard filesystem paths
    isServerVFS   bool              // True for server filesystems
    serverID       string            // Server path for persistence
    userID         string            // User ID for persistence
    onSaveCallback func(map[string]interface{}) error  // Auto-save callback
}
```

#### Key Functions

- `NewVFS(username)` - Creates standard VFS with base structure
- `NewVFSFromMap(username, fs)` - Creates VFS and merges saved changes
- `MergeFromMap(fs)` - Merges filesystem data onto standard base
- `ExtractChanges()` - Extracts only non-standard files for saving
- `SetSaveCallback()` - Sets callback for automatic persistence

#### Deletion Protection

Standard filesystem nodes are protected from deletion:

```go
// Attempting to delete a standard file/directory returns error:
// "cannot delete standard filesystem node: <path>"
```

Standard files can be edited in-session, but changes don't persist (only user-created files are saved).

### File Operations

All file operations trigger automatic persistence:

- **CreateFile** - Creates new file, saves changes
- **CreateDirectory** - Creates new directory, saves changes
- **WriteFile** - Writes content to file, saves changes
- **DeleteNode** - Deletes file/directory (if not standard), saves changes
- **MoveNode** - Moves/renames file (if not standard), saves changes
- **CopyNode** - Copies file/directory, saves changes

### SSH Integration

When a user SSHs into a server:

1. Current VFS is pushed to stack
2. Server's `FileSystem` is loaded from database
3. New VFS is created with standard structure + server changes
4. Save callback is set to persist to server's database record
5. User can create/modify files on the server
6. Changes are automatically saved to server's `FileSystem`

When exiting SSH:

1. Server filesystem changes are saved (via callback)
2. Previous VFS is restored from stack
3. User returns to their local filesystem

### Storage Efficiency

The system only saves **changes** (non-standard files), not the entire filesystem:

**Example User Filesystem:**
```json
{
  "home": {
    "username": {
      "myfile.txt": {
        "content": "Hello World"
      },
      "projects": {
        "project1": {
          "readme.txt": {
            "content": "Project 1"
          }
        }
      }
    }
  }
}
```

Only `myfile.txt` and `projects/` would be saved - not `/bin`, `/usr/bin`, or other standard paths.

### Usage Examples

#### Creating Files Locally

```bash
# User creates a file in their home directory
cd ~
touch notes.txt
edit notes.txt
# File is automatically saved to User.FileSystem
```

#### Creating Files on Server

```bash
# SSH into a server
ssh 1.1.1.1

# Create a file on the server
touch server_notes.txt
edit server_notes.txt
# File is automatically saved to Server.FileSystem
# All players who SSH into this server will see this file
```

#### Standard Files Protection

```bash
# Attempting to delete standard files fails:
rm /bin/ls
# Error: cannot delete standard filesystem node: /bin/ls

# But you can create files in standard directories:
cd /home/username
touch myfile.txt  # This works and is saved
```

### Database Schema

#### User Model
```go
type User struct {
    // ... other fields
    FileSystem map[string]interface{} `gorm:"type:text;serializer:json" json:"file_system"`
}
```

#### Server Model
```go
type Server struct {
    // ... other fields
    FileSystem map[string]interface{} `gorm:"type:text;serializer:json" json:"file_system"`
}
```

### Benefits

1. **Efficient Storage**: Only changes are saved, not entire filesystem structure
2. **Consistent Base**: All users/servers start with same standard structure
3. **Protection**: Standard files cannot be accidentally deleted
4. **Automatic Persistence**: Changes are saved automatically on every operation
5. **Global Server Files**: Server filesystems are shared by all players
6. **Private User Files**: User filesystems are completely private

### Technical Notes

- Standard paths are marked when VFS is created via `markStandardPaths()`
- Path checking uses normalized absolute paths
- Merge operations preserve standard structure while overlaying changes
- Save callbacks are set per-VFS (different for user vs server)
- Stack-based context switching for nested SSH sessions

## Future Enhancements

Consider these potential improvements:

- Tutorial progress tracking (mark steps as complete)
- Interactive tutorial mode (guide users step-by-step)
- Tutorial completion rewards
- More granular tutorial steps with validation
- Tutorial branching based on user actions
- Filesystem permissions (read-only, executable flags)
- File sharing between users
- Filesystem quotas/limits

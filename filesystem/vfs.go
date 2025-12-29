// Package filesystem provides a virtual filesystem implementation for the terminal.sh game.
// It supports directory navigation, file operations, and persistence of user/server filesystem changes.
package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Node represents a file or directory in the virtual filesystem.
type Node struct {
	Name     string            // Name of the file or directory
	IsDir    bool              // True if this is a directory, false if it's a file
	Content  string            // File content (empty for directories)
	Children map[string]*Node  // Child nodes (for directories only)
	Parent   *Node             // Parent node (nil for root)
}

// VFS represents a virtual filesystem with standard Unix-like directory structure.
// It tracks the current directory and supports operations like cd, ls, cat, etc.
type VFS struct {
	Root           *Node
	Current        *Node
	username       string // Store username for home directory
	standardPaths  map[string]bool // Tracks paths that are part of standard filesystem
	isServerVFS   bool   // True if this is a server VFS (for server filesystem persistence)
	serverID       string // Server ID or path for server VFS (for persistence)
	userID         string // User ID for user VFS (for persistence)
	onSaveCallback func(map[string]interface{}) error // Callback to save changes
}

// NewVFS creates a new VFS with standard filesystem structure.
// The filesystem includes /home/{username} as the user's home directory,
// system directories like /bin and /usr/bin, and a README.txt file.
// Returns a VFS instance initialized with the provided username.
func NewVFS(username string) *VFS {
	if username == "" {
		username = "user" // Default fallback
	}

	root := &Node{
		Name:     "/",
		IsDir:    true,
		Children: make(map[string]*Node),
	}

	// Initialize with some default structure
	home := &Node{
		Name:     "home",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   root,
	}
	root.Children["home"] = home

	// Use the actual username instead of hardcoded "user"
	userDir := &Node{
		Name:     username,
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   home,
	}
	home.Children[username] = userDir

	readme := &Node{
		Name:    "README.txt",
		IsDir:   false,
		Content: "Welcome to the SSH Game Server!\n\nThis is a virtual filesystem. Type 'help' to see available commands.",
		Parent:  userDir,
	}
	userDir.Children["README.txt"] = readme

	vfs := &VFS{
		Root:          root,
		Current:       userDir,
		username:      username,
		standardPaths: make(map[string]bool),
		isServerVFS:   false,
	}
	
	// Initialize system directories and commands
	vfs.InitializeSystemCommands()
	
	// Mark all standard paths
	vfs.markStandardPaths()
	
	return vfs
}

// NewVFSFromMap creates a new VFS with standard structure and merges in the provided map.
// This is used for loading persisted server filesystems or user filesystems from the database.
// The provided map is merged on top of the standard filesystem, allowing user-created files
// to be restored while preserving standard system directories.
// Returns a VFS instance and any error that occurred during merging.
func NewVFSFromMap(username string, fs map[string]interface{}) (*VFS, error) {
	vfs := NewVFS(username)
	
	// Merge the provided filesystem data
	if err := vfs.MergeFromMap(fs); err != nil {
		return nil, fmt.Errorf("failed to merge filesystem: %w", err)
	}
	
	return vfs, nil
}

// GetCurrentPath returns the absolute path of the current directory.
func (vfs *VFS) GetCurrentPath() string {
	if vfs.Current == vfs.Root {
		return "/"
	}

	parts := []string{}
	node := vfs.Current
	for node != nil && node != vfs.Root {
		parts = append([]string{node.Name}, parts...)
		node = node.Parent
	}

	return "/" + strings.Join(parts, "/")
}

// ChangeDir changes the current directory to the specified path.
// Supports absolute paths, relative paths, "~" for home directory, "." for current, and ".." for parent.
// Returns an error if the path doesn't exist or is not a directory.
func (vfs *VFS) ChangeDir(path string) error {
	if path == "" || path == "~" {
		// Go to home directory using the stored username
		homePath := "/home/" + vfs.username
		target := vfs.findNode(homePath)
		if target != nil && target.IsDir {
			vfs.Current = target
			return nil
		}
		// Fallback: try to find any user directory under /home
		if home, exists := vfs.Root.Children["home"]; exists {
			// Find the user directory (should be the only child of home, or find by username)
			if userDir, exists := home.Children[vfs.username]; exists && userDir.IsDir {
				vfs.Current = userDir
				return nil
			}
		}
		return fmt.Errorf("cd: home directory not found")
	}

	if path == "/" {
		vfs.Current = vfs.Root
		return nil
	}

	if path == "." {
		return nil
	}

	if path == ".." {
		if vfs.Current.Parent != nil {
			vfs.Current = vfs.Current.Parent
		}
		return nil
	}

	// Handle absolute paths
	if strings.HasPrefix(path, "/") {
		target := vfs.findNode(path)
		if target != nil && target.IsDir {
			vfs.Current = target
			return nil
		}
		return fmt.Errorf("cd: no such file or directory: %s", path)
	}

	// Handle relative paths
	target := vfs.Current.Children[path]
	if target != nil && target.IsDir {
		vfs.Current = target
		return nil
	}

	return fmt.Errorf("cd: no such file or directory: %s", path)
}

// ListDir returns a list of nodes in the current directory, excluding hidden files.
func (vfs *VFS) ListDir() []*Node {
	return vfs.ListDirWithOptions(false)
}

// ListDirWithOptions lists directory contents with options.
// If showAll is true, includes hidden files (starting with ".").
func (vfs *VFS) ListDirWithOptions(showAll bool) []*Node {
	if !vfs.Current.IsDir {
		return []*Node{}
	}

	nodes := make([]*Node, 0, len(vfs.Current.Children))
	for _, child := range vfs.Current.Children {
		// Filter hidden files unless showAll is true
		if !showAll && strings.HasPrefix(child.Name, ".") {
			continue
		}
		nodes = append(nodes, child)
	}
	return nodes
}

// ReadFile reads the content of a file in the current directory.
// Returns the file content and an error if the file doesn't exist or is a directory.
func (vfs *VFS) ReadFile(name string) (string, error) {
	if file, exists := vfs.Current.Children[name]; exists && !file.IsDir {
		return file.Content, nil
	}
	return "", fmt.Errorf("cat: %s: no such file or directory", name)
}

func (vfs *VFS) findNode(path string) *Node {
	path = filepath.Clean(path)
	parts := strings.Split(strings.Trim(path, "/"), "/")

	current := vfs.Root
	for _, part := range parts {
		if part == "" {
			continue
		}
		if child, exists := current.Children[part]; exists {
			current = child
		} else {
			return nil
		}
	}
	return current
}

// CreateFile creates a new empty file in the current directory.
// Returns an error if a file or directory with the same name already exists.
// Triggers the save callback if set to persist the change.
func (vfs *VFS) CreateFile(name string) error {
	if _, exists := vfs.Current.Children[name]; exists {
		return fmt.Errorf("file already exists: %s", name)
	}
	
	file := &Node{
		Name:    name,
		IsDir:   false,
		Content: "",
		Parent:  vfs.Current,
	}
	vfs.Current.Children[name] = file
	
	// Trigger save callback if set
	if vfs.onSaveCallback != nil {
		changes := vfs.ExtractChanges()
		if err := vfs.onSaveCallback(changes); err != nil {
			// Log error but don't fail the create operation
		}
	}
	
	return nil
}

// CreateDirectory creates a new directory in the current directory.
// Returns an error if a file or directory with the same name already exists.
// Triggers the save callback if set to persist the change.
func (vfs *VFS) CreateDirectory(name string) error {
	if _, exists := vfs.Current.Children[name]; exists {
		return fmt.Errorf("directory already exists: %s", name)
	}
	
	dir := &Node{
		Name:     name,
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   vfs.Current,
	}
	vfs.Current.Children[name] = dir
	
	// Trigger save callback if set
	if vfs.onSaveCallback != nil {
		changes := vfs.ExtractChanges()
		if err := vfs.onSaveCallback(changes); err != nil {
			// Log error but don't fail the create operation
		}
	}
	
	return nil
}

// DeleteNode deletes a file or directory in the current directory.
// If recursive is true, deletes directories even if they contain files.
// Returns an error if trying to delete a standard filesystem node or if the directory is not empty (when recursive is false).
// Triggers the save callback if set to persist the change.
func (vfs *VFS) DeleteNode(name string, recursive bool) error {
	node, exists := vfs.Current.Children[name]
	if !exists {
		return fmt.Errorf("no such file or directory: %s", name)
	}
	
	// Check if this is a standard filesystem node
	fullPath := vfs.GetCurrentPath() + "/" + name
	if strings.HasSuffix(fullPath, "/") {
		fullPath = fullPath[:len(fullPath)-1]
	}
	if vfs.isStandardPath(fullPath) {
		return fmt.Errorf("cannot delete standard filesystem node: %s", name)
	}
	
	// Check if directory contains standard paths
	if node.IsDir && recursive {
		if err := vfs.checkDirectoryForStandardPaths(node, fullPath); err != nil {
			return err
		}
	}
	
	if node.IsDir && len(node.Children) > 0 && !recursive {
		return fmt.Errorf("cannot remove directory '%s': directory not empty", name)
	}
	
	delete(vfs.Current.Children, name)
	
	// Trigger save callback if set
	if vfs.onSaveCallback != nil {
		changes := vfs.ExtractChanges()
		if err := vfs.onSaveCallback(changes); err != nil {
			// Log error but don't fail the delete operation
			// The file is already deleted in memory
		}
	}
	
	return nil
}

// CopyNode copies a file or directory from src to dest in the current directory.
// The destination must not already exist. Returns an error if source doesn't exist or destination exists.
// Triggers the save callback if set to persist the change.
func (vfs *VFS) CopyNode(src, dest string) error {
	srcNode := vfs.Current.Children[src]
	if srcNode == nil {
		return fmt.Errorf("source not found: %s", src)
	}
	
	if _, exists := vfs.Current.Children[dest]; exists {
		return fmt.Errorf("destination already exists: %s", dest)
	}
	
	// Create a copy
	newNode := &Node{
		Name:     dest,
		IsDir:    srcNode.IsDir,
		Content:  srcNode.Content,
		Parent:   vfs.Current,
	}
	
	if srcNode.IsDir {
		newNode.Children = make(map[string]*Node)
		// Recursively copy children (simplified)
		for name, child := range srcNode.Children {
			childCopy := &Node{
				Name:     name,
				IsDir:    child.IsDir,
				Content:  child.Content,
				Parent:   newNode,
			}
			if child.IsDir {
				childCopy.Children = make(map[string]*Node)
			}
			newNode.Children[name] = childCopy
		}
	}
	
	vfs.Current.Children[dest] = newNode
	
	// Trigger save callback if set
	if vfs.onSaveCallback != nil {
		changes := vfs.ExtractChanges()
		if err := vfs.onSaveCallback(changes); err != nil {
			// Log error but don't fail the copy operation
		}
	}
	
	return nil
}

// MoveNode moves or renames a file or directory from src to dest in the current directory.
// Cannot move standard filesystem nodes. Returns an error if source doesn't exist or destination exists.
// Triggers the save callback if set to persist the change.
func (vfs *VFS) MoveNode(src, dest string) error {
	srcNode := vfs.Current.Children[src]
	if srcNode == nil {
		return fmt.Errorf("source not found: %s", src)
	}
	
	// Check if source is a standard path
	srcPath := vfs.GetCurrentPath() + "/" + src
	if strings.HasSuffix(srcPath, "/") {
		srcPath = srcPath[:len(srcPath)-1]
	}
	if vfs.isStandardPath(srcPath) {
		return fmt.Errorf("cannot move standard filesystem node: %s", src)
	}
	
	if _, exists := vfs.Current.Children[dest]; exists {
		return fmt.Errorf("destination already exists: %s", dest)
	}
	
	// Rename/move
	srcNode.Name = dest
	vfs.Current.Children[dest] = srcNode
	delete(vfs.Current.Children, src)
	
	// Trigger save callback if set
	if vfs.onSaveCallback != nil {
		changes := vfs.ExtractChanges()
		if err := vfs.onSaveCallback(changes); err != nil {
			// Log error but don't fail the move operation
		}
	}
	
	return nil
}

// WriteFile writes content to a file in the current directory.
// Returns an error if the file doesn't exist or is a directory.
// Triggers the save callback if set to persist the change.
func (vfs *VFS) WriteFile(name, content string) error {
	file, exists := vfs.Current.Children[name]
	if !exists {
		return fmt.Errorf("file not found: %s", name)
	}
	
	if file.IsDir {
		return fmt.Errorf("cannot write to directory: %s", name)
	}
	
	file.Content = content
	
	// Trigger save callback if set
	if vfs.onSaveCallback != nil {
		changes := vfs.ExtractChanges()
		if err := vfs.onSaveCallback(changes); err != nil {
			// Log error but don't fail the write operation
		}
	}
	
	return nil
}

// RenameHomeDirectory renames the user's home directory (used when username changes).
// Updates the home directory path from /home/{oldUsername} to /home/{newUsername}.
// Returns an error if the new username is invalid or the directory already exists.
func (vfs *VFS) RenameHomeDirectory(newUsername string) error {
	if newUsername == "" {
		return fmt.Errorf("invalid username")
	}

	// Get the home directory node
	home, exists := vfs.Root.Children["home"]
	if !exists {
		return fmt.Errorf("home directory not found")
	}

	// Check if new username already exists
	if _, exists := home.Children[newUsername]; exists {
		return fmt.Errorf("directory /home/%s already exists", newUsername)
	}

	// Find the current user directory
	oldUserDir, exists := home.Children[vfs.username]
	if !exists {
		return fmt.Errorf("current user directory /home/%s not found", vfs.username)
	}

	// Rename the directory
	oldUserDir.Name = newUsername
	home.Children[newUsername] = oldUserDir
	delete(home.Children, vfs.username)

	// Update the stored username
	vfs.username = newUsername

	// If we're currently in the old home directory, update Current to point to the renamed one
	if vfs.Current == oldUserDir {
		vfs.Current = oldUserDir
	} else {
		// Check if current path is under the old home directory and update it
		// This is a simplified approach - we'll just make sure Current still points to the right node
		// The node reference itself doesn't change, just the name and map key
	}

	return nil
}

// GetUsername returns the current username associated with this VFS.
func (vfs *VFS) GetUsername() string {
	return vfs.username
}

// InitializeSystemCommands creates /bin and /usr/bin directories and populates them with command files.
// This sets up the standard command structure used by the help system.
func (vfs *VFS) InitializeSystemCommands() {
	// Create /bin directory
	binDir := &Node{
		Name:     "bin",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   vfs.Root,
	}
	vfs.Root.Children["bin"] = binDir
	
	// Create /usr directory
	usrDir := &Node{
		Name:     "usr",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   vfs.Root,
	}
	vfs.Root.Children["usr"] = usrDir
	
	// Create /usr/bin directory
	usrBinDir := &Node{
		Name:     "bin",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   usrDir,
	}
	usrDir.Children["bin"] = usrBinDir
	
	// Define system commands with their descriptions
	systemCommands := map[string]string{
		"ls":              "List directory contents",
		"cd":              "Change directory",
		"pwd":             "Print working directory",
		"cat":             "Display file contents",
		"touch":           "Create a new file",
		"mkdir":           "Create a new directory",
		"rm":              "Delete file or directory",
		"cp":              "Copy files/folders",
		"mv":              "Move or rename files/folders",
		"edit":            "Edit a file",
		"clear":           "Clear the screen",
		"help":            "Show available commands",
		"whoami":          "Display current username",
		"name":            "Change username",
		"ifconfig":        "Show network interfaces",
		"scan":            "Scan internet or IP",
		"ssh":             "Connect to a server",
		"exit":            "Disconnect from server",
		"server":          "Show current server info",
		"createServer":    "Create a new server",
		"createLocalServer": "Create local server",
		"get":             "Download tool from server",
		"tools":           "List owned tools",
		"exploited":       "List exploited servers",
		"wallet":          "Show wallet balance",
		"crypto_miner":    "Start mining",
		"stop_mining":     "Stop mining",
		"miners":          "List active miners",
		"userinfo":        "Show user information",
		"info":            "Display browser/client info",
	}
	
	// Create command files in /bin
	for cmd, desc := range systemCommands {
		cmdFile := &Node{
			Name:    cmd,
			IsDir:   false,
			Content: desc,
			Parent:  binDir,
		}
		binDir.Children[cmd] = cmdFile
	}
}

// GetCommandDescription retrieves the description of a command from the filesystem.
// Checks both /bin and /usr/bin directories. Returns the description and an error if not found.
func (vfs *VFS) GetCommandDescription(cmdName string) (string, error) {
	// Check in /bin first
	binPath := "/bin/" + cmdName
	binNode := vfs.findNode(binPath)
	if binNode != nil && !binNode.IsDir {
		return binNode.Content, nil
	}
	
	// Check in /usr/bin
	usrBinPath := "/usr/bin/" + cmdName
	usrBinNode := vfs.findNode(usrBinPath)
	if usrBinNode != nil && !usrBinNode.IsDir {
		return usrBinNode.Content, nil
	}
	
	return "", fmt.Errorf("command not found: %s", cmdName)
}

// ListCommands lists all available commands from /bin and /usr/bin.
// Returns two slices: bin commands and usrBin commands.
func (vfs *VFS) ListCommands() ([]string, []string) {
	var binCommands []string
	var usrBinCommands []string
	
	// List /bin commands
	binNode := vfs.findNode("/bin")
	if binNode != nil && binNode.IsDir {
		for name := range binNode.Children {
			if !binNode.Children[name].IsDir {
				binCommands = append(binCommands, name)
			}
		}
	}
	
	// List /usr/bin commands
	usrBinNode := vfs.findNode("/usr/bin")
	if usrBinNode != nil && usrBinNode.IsDir {
		for name := range usrBinNode.Children {
			if !usrBinNode.Children[name].IsDir {
				usrBinCommands = append(usrBinCommands, name)
			}
		}
	}
	
	return binCommands, usrBinCommands
}

// AddUserCommand adds a command file to /usr/bin (for user-acquired tools).
// Returns an error if /usr/bin doesn't exist or the command already exists.
func (vfs *VFS) AddUserCommand(cmdName, description string) error {
	usrBinNode := vfs.findNode("/usr/bin")
	if usrBinNode == nil || !usrBinNode.IsDir {
		return fmt.Errorf("/usr/bin directory not found")
	}
	
	// Check if command already exists
	if _, exists := usrBinNode.Children[cmdName]; exists {
		return fmt.Errorf("command %s already exists", cmdName)
	}
	
	cmdFile := &Node{
		Name:    cmdName,
		IsDir:   false,
		Content: description,
		Parent:  usrBinNode,
	}
	usrBinNode.Children[cmdName] = cmdFile
	
	return nil
}

// markStandardPaths recursively marks all paths in the standard filesystem
func (vfs *VFS) markStandardPaths() {
	vfs.markNodePaths(vfs.Root, "/")
}

// markNodePaths recursively marks paths starting from a node
func (vfs *VFS) markNodePaths(node *Node, path string) {
	if path != "/" {
		vfs.standardPaths[path] = true
	}
	
	for name, child := range node.Children {
		childPath := path
		if childPath == "/" {
			childPath = "/" + name
		} else {
			childPath = path + "/" + name
		}
		vfs.markNodePaths(child, childPath)
	}
}

// isStandardPath checks if a path is part of the standard filesystem
func (vfs *VFS) isStandardPath(path string) bool {
	// Normalize path
	path = filepath.Clean(path)
	if path == "." {
		path = vfs.GetCurrentPath()
	}
	if !strings.HasPrefix(path, "/") {
		path = vfs.GetCurrentPath() + "/" + path
	}
	path = filepath.Clean(path)
	
	return vfs.standardPaths[path]
}

// checkDirectoryForStandardPaths checks if a directory contains standard paths
func (vfs *VFS) checkDirectoryForStandardPaths(node *Node, basePath string) error {
	for name, child := range node.Children {
		childPath := basePath + "/" + name
		if vfs.isStandardPath(childPath) {
			return fmt.Errorf("cannot delete directory containing standard filesystem node: %s", childPath)
		}
		if child.IsDir {
			if err := vfs.checkDirectoryForStandardPaths(child, childPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExtractChanges extracts only non-standard files/directories from the VFS.
// Returns a map structure with only user-created/modified content, suitable for persistence.
// Standard filesystem nodes (like /bin, /usr/bin) are excluded from the result.
func (vfs *VFS) ExtractChanges() map[string]interface{} {
	changes := make(map[string]interface{})
	vfs.extractChangesFromNode(vfs.Root, "/", changes)
	return changes
}

// extractChangesFromNode recursively extracts non-standard nodes
func (vfs *VFS) extractChangesFromNode(node *Node, path string, changes map[string]interface{}) {
	for name, child := range node.Children {
		childPath := path
		if childPath == "/" {
			childPath = "/" + name
		} else {
			childPath = path + "/" + name
		}
		
		// Normalize path for comparison
		normalizedPath := filepath.Clean(childPath)
		
		// Check if this is a standard path
		isStandard := vfs.isStandardPath(normalizedPath)
		
		// Get relative path from root for building structure
		relPath := strings.TrimPrefix(normalizedPath, "/")
		parts := strings.Split(relPath, "/")
		
		// Navigate to the parent directory in changes map
		current := changes
		for i := 0; i < len(parts)-1; i++ {
			part := parts[i]
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}
			current = current[part].(map[string]interface{})
		}
		
		// Now handle the current node
		if !isStandard {
			// Non-standard node - add it to changes
			// Use the child's name directly (not derived from path parts)
			if child.IsDir {
				// Create directory entry
				if _, exists := current[name]; !exists {
					current[name] = make(map[string]interface{})
				}
				// Recurse into directory to find all non-standard children
				dirMap := current[name].(map[string]interface{})
				vfs.extractChangesFromNode(child, childPath, dirMap)
			} else {
				// Add file with content
				current[name] = map[string]interface{}{
					"content": child.Content,
				}
			}
		} else if child.IsDir {
			// Standard directory - recurse to find non-standard children
			// Build path structure in changes map for recursion
			// Use the child's name directly (not derived from path parts)
			if _, exists := current[name]; !exists {
				current[name] = make(map[string]interface{})
			}
			dirMap := current[name].(map[string]interface{})
			vfs.extractChangesFromNode(child, childPath, dirMap)
		}
		// If it's a standard file, we skip it entirely
	}
}

// MergeFromMap merges a map structure into the VFS, overlaying on top of standard filesystem.
// This is used to restore persisted filesystem changes from the database.
// Returns an error if the merge operation fails due to conflicts.
func (vfs *VFS) MergeFromMap(fs map[string]interface{}) error {
	if fs == nil || len(fs) == 0 {
		return nil
	}
	return vfs.mergeIntoNode(vfs.Root, "/", fs)
}

// mergeIntoNode recursively merges map structure into VFS nodes
func (vfs *VFS) mergeIntoNode(parent *Node, parentPath string, data map[string]interface{}) error {
	for key, value := range data {
		childPath := parentPath
		if childPath == "/" {
			childPath = "/" + key
		} else {
			childPath = parentPath + "/" + key
		}
		
		switch v := value.(type) {
		case map[string]interface{}:
			// Check if it's a file with content or a directory
			if content, isFile := v["content"].(string); isFile {
				// It's a file
				// Check if node exists
				if existing, exists := parent.Children[key]; exists {
					if existing.IsDir {
						return fmt.Errorf("cannot merge file into existing directory: %s", childPath)
					}
					existing.Content = content
				} else {
					// Create new file
					file := &Node{
						Name:    key,
						IsDir:   false,
						Content: content,
						Parent:  parent,
					}
					parent.Children[key] = file
				}
			} else {
				// It's a directory
				var dirNode *Node
				if existing, exists := parent.Children[key]; exists {
					if !existing.IsDir {
						return fmt.Errorf("cannot merge directory into existing file: %s", childPath)
					}
					dirNode = existing
				} else {
					// Create new directory
					dirNode = &Node{
						Name:     key,
						IsDir:    true,
						Children: make(map[string]*Node),
						Parent:   parent,
					}
					parent.Children[key] = dirNode
				}
				
				// Recurse into directory
				if err := vfs.mergeIntoNode(dirNode, childPath, v); err != nil {
					return err
				}
			}
		case string:
			// Direct string value (legacy format)
			if existing, exists := parent.Children[key]; exists {
				if existing.IsDir {
					return fmt.Errorf("cannot merge file into existing directory: %s", childPath)
				}
				existing.Content = v
			} else {
				file := &Node{
					Name:    key,
					IsDir:   false,
					Content: v,
					Parent:  parent,
				}
				parent.Children[key] = file
			}
		}
	}
	return nil
}

// SetSaveCallback sets a callback function to be called when filesystem changes occur.
// The callback receives the extracted changes map and should persist them to storage.
// The callback is invoked automatically after create, delete, write, copy, and move operations.
func (vfs *VFS) SetSaveCallback(callback func(map[string]interface{}) error) {
	vfs.onSaveCallback = callback
}

// SetServerID sets the server ID/path for server VFS persistence.
// Marks this VFS as a server filesystem and stores the server identifier.
func (vfs *VFS) SetServerID(serverID string) {
	vfs.isServerVFS = true
	vfs.serverID = serverID
}

// SetUserID sets the user ID for user VFS persistence.
// Marks this VFS as a user filesystem (not a server filesystem) and stores the user identifier.
func (vfs *VFS) SetUserID(userID string) {
	vfs.isServerVFS = false
	vfs.userID = userID
}


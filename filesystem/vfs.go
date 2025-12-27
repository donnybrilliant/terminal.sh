package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Node struct {
	Name     string
	IsDir    bool
	Content  string
	Children map[string]*Node
	Parent   *Node
}

type VFS struct {
	Root     *Node
	Current  *Node
	username string // Store username for home directory
}

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
		Root:     root,
		Current:  userDir,
		username: username,
	}
	
	// Initialize system directories and commands
	vfs.InitializeSystemCommands()
	
	return vfs
}

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

func (vfs *VFS) ListDir() []*Node {
	return vfs.ListDirWithOptions(false)
}

// ListDirWithOptions lists directory contents with options
// showAll: if true, include hidden files (starting with .)
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

// CreateFile creates a new file
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
	return nil
}

// CreateDirectory creates a new directory
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
	return nil
}

// DeleteNode deletes a file or directory
func (vfs *VFS) DeleteNode(name string, recursive bool) error {
	node, exists := vfs.Current.Children[name]
	if !exists {
		return fmt.Errorf("no such file or directory: %s", name)
	}
	
	if node.IsDir && len(node.Children) > 0 && !recursive {
		return fmt.Errorf("cannot remove directory '%s': directory not empty", name)
	}
	
	delete(vfs.Current.Children, name)
	return nil
}

// CopyNode copies a file or directory
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
	return nil
}

// MoveNode moves or renames a file or directory
func (vfs *VFS) MoveNode(src, dest string) error {
	srcNode := vfs.Current.Children[src]
	if srcNode == nil {
		return fmt.Errorf("source not found: %s", src)
	}
	
	if _, exists := vfs.Current.Children[dest]; exists {
		return fmt.Errorf("destination already exists: %s", dest)
	}
	
	// Rename/move
	srcNode.Name = dest
	vfs.Current.Children[dest] = srcNode
	delete(vfs.Current.Children, src)
	return nil
}

// WriteFile writes content to a file
func (vfs *VFS) WriteFile(name, content string) error {
	file, exists := vfs.Current.Children[name]
	if !exists {
		return fmt.Errorf("file not found: %s", name)
	}
	
	if file.IsDir {
		return fmt.Errorf("cannot write to directory: %s", name)
	}
	
	file.Content = content
	return nil
}

// RenameHomeDirectory renames the user's home directory (used when username changes)
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

// GetUsername returns the current username
func (vfs *VFS) GetUsername() string {
	return vfs.username
}

// InitializeSystemCommands creates /bin and /usr/bin directories and populates them with command files
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

// GetCommandDescription retrieves the description of a command from the filesystem
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

// ListCommands lists all available commands from /bin and /usr/bin
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

// AddUserCommand adds a command file to /usr/bin (for user-acquired tools)
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


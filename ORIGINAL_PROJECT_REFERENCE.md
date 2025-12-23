# Original Node.js Project Reference

This document describes the functionality and data structures from the original Node.js SSH game project. Use this as reference when implementing features in the Go version.

## Table of Contents

1. [Data Structures](#data-structures)
2. [Commands](#commands)
3. [Game Features](#game-features)
4. [Network & SSH System](#network--ssh-system)
5. [User System](#user-system)
6. [Tools System](#tools-system)
7. [Filesystem](#filesystem)

---

## Data Structures

### User Object

```javascript
{
  id: "uuid",
  username: "string",
  password: "hashed_password",
  ip: "external_ip",           // e.g., "1.2.3.4"
  localIP: "local_ip",         // e.g., "10.0.0.5"
  mac: "mac_address",
  home: {},                    // User's home directory data
  level: 0,
  experience: 0,
  resources: {
    cpu: 200,
    bandwidth: 300,
    ram: 24
  },
  wallet: {
    crypto: 15,
    data: 1200
  },
  tools: [],                   // Array of tool objects (see Tools section)
  achievements: [],
  inventory: {
    items: [],
    currency: 500
  },
  localNetwork: {},            // User's local network of servers
  exploitedServers: {},        // Map of exploited servers by path
  activeMiners: {}             // Active mining sessions
}
```

### Server Object (in internet.json)

```javascript
{
  ip: "string",                // External IP or special name like "repo"
  localIP: "string",           // Local IP like "10.0.0.1"
  securityLevel: 100,
  resources: {
    cpu: 10000,
    bandwidth: 50000,
    ram: 1048
  },
  wallet: {
    crypto: 10000,
    data: 50000000
  },
  tools: ["tool_name1", "tool_name2"],  // Array of tool names
  connectedIPs: ["ip1", "ip2"],         // Connected servers
  services: [
    {
      name: "ssh",
      description: "Secure Shell",
      port: 22,
      vulnerable: true,
      level: 10,
      vulnerabilities: [
        {
          type: "remote_code_execution",
          level: 20
        },
        {
          type: "buffer_overflow",
          level: 30
        }
      ]
    }
  ],
  roles: [
    {
      role: "admin",
      level: 100
    }
  ],
  logs: [],
  fileSystem: {
    home: {
      admin: {
        "README.txt": {
          content: "Admin home directory."
        }
      }
    }
  },
  localNetwork: {},            // Nested servers (servers within this server)
  usedResources: {             // Runtime: resources currently in use
    cpu: 0,
    bandwidth: 0,
    ram: 0
  },
  activeMiners: {}             // Runtime: active miners on this server
}
```

### Tool Object

```javascript
{
  name: "tool_name",           // e.g., "ssh_exploit", "crypto_miner"
  function: "Description",
  resources: {
    cpu: 20,
    bandwidth: 0.3,
    ram: 8
  },
  exploits: [                  // Optional: vulnerabilities this tool can exploit
    {
      type: "remote_code_execution",
      level: 20
    }
  ],
  services: "ssh",             // Optional: service this tool targets
  special: "special property", // Optional: e.g., "Generates passive income over time."
  isPatch: false               // Optional: if true, patches/upgrades another tool
}
```

---

## Commands

### Filesystem Commands

- `ls [-l]` - List directory contents
- `cd <directory>` - Change directory (supports `.`, `..`, `~`, absolute paths)
- `pwd` - Print working directory
- `cat <filename>` - Display file contents
- `touch <filename>` - Create a new file
- `mkdir <dirname>` - Create a new directory
- `rm <filename>` - Delete file
- `rm -r <folder>` - Delete folder recursively
- `cp <src> <dest>` - Copy files/folders
- `mv <src> <dest>` - Move or rename files/folders
- `edit|vi|nano <filename>` - Edit a file (enters edit mode)
  - In edit mode: `:save` to save, `:exit` to exit

### System Commands

- `clear` - Clear the screen
- `help` - Show available commands
- `whoami` - Display current username
- `info` - Display browser/client info
- `name <newName>` - Change username
- `login <username> <password>` - Login
- `logout` - Logout

### Network Commands

- `scan` - Scan internet for IP addresses (when not in SSH mode)
- `scan <targetIP>` - Scan target IP for services and vulnerabilities
- `scan` (in SSH mode) - Scan connected IPs on current server
- `ifconfig` - Show network interfaces (user's IP, localIP, MAC)
- `ssh <targetIP>` - Connect to a server via SSH
  - Supports nested SSH (SSH into servers within servers)
  - Creates parent/child session hierarchy

### Game Commands

- `get <targetIP> <toolName>` - Download a tool from a server
- `tools` - List user's available tools
- `exploited` - List exploited servers
- `wallet` - Show wallet balance (crypto and data)
- `userinfo` - Show user information (level, experience, achievements, etc.)
- `miners` - List active miners
- `server` - Show hardware info of current server (in SSH mode)
- `createServer` - Create a new server
- `createLocalServer` - Create a local server on current connection

### Tool-Specific Commands

Commands that are only available when the user has the corresponding tool:

- `password_cracker <targetIP>` - Crack passwords on a server
- `password_sniffer <targetIP>` - Sniff and crack passwords from user roles
- `ssh_exploit <targetIP>` - Exploit SSH vulnerabilities
- `crypto_miner <targetIP>` - Start mining cryptocurrency
- `stop_mining <targetIP>` - Stop mining
- `lan_sniffer <targetIP>` - Sniff for local network connections
- `packet_capture <targetIP>` - Capture network packets
- `packet_decoder <targetIP>` - Decode captured packets
- `user_enum <targetIP>` - Enumerate users and roles
- `rootkit <targetIP>` - Install rootkit for hidden access
- `exploit_kit <targetIP>` - Exploit multiple vulnerability types
- `sql_injector <targetIP>` - Perform SQL injection attacks
- `xss_exploit <targetIP>` - Exploit XSS vulnerabilities

### Chat Commands

- `chat` - Enter chat mode
  - In chat mode: `exit` to leave, `join <room>` to join a room
- Commands prefixed with `/` in chat mode have special meaning

### Fun/Animation Commands

- `matrix` - Start Matrix animation
- `hack` - Simulate hacking animation
- `loadtest` - Load test terminal
- `chars` - Character test

---

## Game Features

### Authentication System

- Users can register by attempting to login with new credentials
- Passwords are hashed using bcrypt
- JWT tokens for session management
- Guest mode available (read-only access)

### Network Scanning

1. **Internet Scan** (`scan` with no args):
   - Returns list of all top-level IP addresses in internet.json
   - Shows available servers to connect to

2. **IP Scan** (`scan <targetIP>`):
   - Scans a specific IP for:
     - Services (SSH, HTTP, FTP, etc.)
     - Vulnerabilities per service
     - Tools available on the server
     - Resource capacity
     - Security level

3. **Local Network Scan** (`scan` in SSH mode):
   - Scans `connectedIPs` on the current server
   - Used to discover nested servers

### SSH System

- **Nested SSH**: Users can SSH into servers, then SSH into servers within those servers
- **Session Hierarchy**: Tracks parent/child relationships
- **Path-based Exploitation**: Servers accessed via nested SSH use path notation:
  - Top level: `"1.1.1.1"`
  - Nested: `"1.1.1.1.localNetwork.10.0.0.5"`
- **Filesystem Switching**: Each SSH session has its own filesystem context
- **Parent Tracking**: Commands need to know parent IP for nested connections

### Exploitation System

1. **Tool Matching**: Tools must match vulnerability types and levels
   - Tool exploit level must be >= vulnerability level
   - Tool must support the vulnerability type

2. **Exploitation Process**:
   - User runs tool command (e.g., `ssh_exploit <targetIP>`)
   - System checks if tool matches vulnerabilities
   - If successful, server is marked as exploited in `user.exploitedServers`
   - Exploitation path is stored: `"targetIP.localNetwork.nestedIP"`

3. **Exploited Servers**:
   - Stored in user object: `exploitedServers[exploitationPath][serviceName] = [exploits]`
   - Users can only SSH into exploited servers
   - Multiple exploits can be used on the same service

### Mining System

- **Crypto Mining**: Users can install crypto miners on exploited servers
- **Resource Requirements**: Miners consume CPU, bandwidth, RAM
- **Resource Checking**: System verifies available resources before starting
- **Passive Income**: Miners generate cryptocurrency over time
- **Multiple Miners**: Users can run multiple miners on different servers
- **Active Miners Tracking**: 
  - User object: `activeMiners[targetIP] = { startTime, resourceUsage }`
  - Server object: `activeMiners[userIP] = { startTime, resourceUsage }`

### Tool System

- **Tool Repository**: "repo" server contains all available tools
- **Downloading Tools**: Use `get <targetIP> <toolName>` to download
- **Tool Storage**: Tools stored in `user.tools[]` array
- **Tool Merging**: Patches (tools with `isPatch: true`) upgrade existing tools
- **Tool Execution**: Tools add new commands to the command processor
- **Resource Costs**: Tools consume resources when used

### Server Creation

- **Default Template**: Uses `serverConfig.json` template
- **IP Generation**: Generates unique external and local IPs
- **Randomization**: Randomizes security levels, resources, vulnerabilities
- **Local Servers**: Can create servers within other servers (nested)

### Filesystem

- **Dual Filesystems**: 
  - User's local filesystem (when not in SSH)
  - Target server's filesystem (when in SSH mode)
- **Path Stack**: Tracks current directory using path arrays
- **User Home Directories**: Each user has `/home/users/<username>/`
- **File Objects**: Files stored as `{ content: "file contents" }`
- **Directory Objects**: Directories stored as objects with nested structure

---

## Network & SSH System

### IP Addresses

- **External IPs**: Public-facing IPs (e.g., "1.1.1.1", "2.2.2.2")
- **Local IPs**: Private network IPs (e.g., "10.0.0.1", "10.0.0.2")
- **Special IPs**: 
  - "repo" - Tool repository server
  - User IPs - Generated unique IPs for each user

### SSH Session Structure

```javascript
{
  targetIP: "1.1.1.1",
  parents: [
    { targetIP: "parent1", ... },
    { targetIP: "parent2", ... }
  ],
  // Other session data
}
```

### Exploitation Path Format

- Top-level: `"1.1.1.1"`
- One level deep: `"1.1.1.1.localNetwork.10.0.0.5"`
- Multiple levels: `"1.1.1.1.localNetwork.10.0.0.5.localNetwork.10.0.0.10"`

---

## User System

### User Registration

- Automatic registration on first login
- Validates username (cannot be "guest")
- Generates unique IP, localIP, and MAC address
- Initial resources and wallet balance

### User Progression

- **Level**: Increases with experience
- **Experience**: Gained from actions (exploiting, mining, etc.)
- **Achievements**: List of unlocked achievements
- **Resources**: User's computational resources

### User Wallet

- **Crypto**: Cryptocurrency (gained from mining, spent on tools/items)
- **Data**: Data currency (gained from various activities)

---

## Tools System

### Available Tools

**Exploitation Tools:**
- `password_cracker` - Basic password cracking
- `pass_patch` - Upgrade for password_cracker
- `password_sniffer` - Sniff and crack passwords from roles
- `ssh_exploit` - Exploit SSH vulnerabilities
- `ssh_patch` - Upgrade for ssh_exploit
- `exploit_kit` - Multi-vulnerability exploitation
- `advanced_exploit_kit` - Advanced multi-vulnerability exploitation
- `sql_injector` - SQL injection attacks
- `xss_exploit` - XSS exploitation

**Information Gathering:**
- `user_enum` - Enumerate users and roles
- `lan_sniffer` - Discover local network connections
- `packet_capture` - Capture network packets
- `packet_decoder` - Decode captured packets

**Persistence & Control:**
- `rootkit` - Install hidden backdoor access

**Income Generation:**
- `crypto_miner` - Mine cryptocurrency (passive income)

### Tool Patches

- Tools with `isPatch: true` upgrade existing tools
- Patches increase exploit levels or add capabilities
- Example: `pass_patch` upgrades `password_cracker` from level 10 to 20

---

## Filesystem

### Structure

```
filesystem.json (base structure)
├── home/
│   └── users/
│       ├── guest/
│       └── <username>/
│           ├── README.txt
│           ├── bin/          # Downloaded tools go here
│           └── ...
└── etc/
    └── config/
```

### File Representation

```javascript
{
  "filename.txt": {
    content: "File contents here"
  }
}
```

### Directory Representation

```javascript
{
  "dirname": {
    "nested_file.txt": {
      content: "..."
    },
    "nested_dir": {
      // More nested content
    }
  }
}
```

### Path Handling

- Absolute paths start with `/`
- Relative paths: `.`, `..`, `~` (home directory)
- Path stack tracks current directory as array: `["home", "users", "username"]`
- Separate path stacks for local filesystem vs SSH filesystem

---

## Important Implementation Notes

1. **JSON Storage**: Original project uses JSON files for persistence
   - `data/users.json` - User data
   - `data/internet.json` - Server network data
   - `data/tools.json` - Tool definitions
   - `data/filesystem.json` - Base filesystem structure
   - `data/store.json` - Store/items data

2. **Socket.IO**: Real-time communication via WebSockets
   - Commands emit events, server responds with results
   - Bidirectional communication for game state updates

3. **Session Management**: 
   - User sessions tracked in memory
   - SSH sessions maintain parent/child relationships
   - Filesystem state maintained per session

4. **Resource Management**:
   - Tools consume resources (CPU, bandwidth, RAM)
   - Servers track used vs available resources
   - Mining requires available resources on target server

5. **Security Levels**:
   - Higher security = harder to exploit
   - Tools must match or exceed vulnerability levels
   - Security affects exploit success rates

---

## Future Enhancements (Not in Original)

Consider these when extending the Go version:

- Database backend (currently JSON files)
- Multiplayer chat rooms
- Alliance/guild system
- More tool types
- Persistent mining income
- Server management/upgrades
- More vulnerability types
- Web interface options

---

*This reference is based on the Node.js implementation in `/Users/daniel/Downloads/ssh-game/`*


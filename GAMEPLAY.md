# terminal.sh Gameplay Guide

Welcome to terminal.sh, a terminal-based hacking simulation game! This guide will help you learn how to play, from basic commands to advanced exploitation techniques.

## What is terminal.sh?

terminal.sh is a hacking simulation game where you:
- Explore a virtual network by scanning and exploiting servers
- Download and use hacking tools to break into systems
- Mine cryptocurrency for passive income
- Upgrade your tools with patches
- Shop for new tools and upgrades
- Chat with other players in real-time
- Build your skills through tutorials

## Getting Started

### Connecting to the Game

#### SSH Connection

Connect using an SSH client:

```bash
ssh -p 2222 <username>@your-server-ip
```

**Authentication:**
- The server uses password authentication
- **Auto-registration**: Any username/password combination will automatically create a new account on first login
- After registration, use the same credentials to log in
- Example: `ssh -p 2222 daniel@localhost` (password can be anything on first login)

#### Web Connection

Open your browser and navigate to:

```
http://your-server-ip:8080
```

The browser will automatically connect via WebSocket and display the same terminal interface as SSH.

### First Steps

1. **Check your information:**
   ```bash
   userinfo    # View your stats, level, and resources
   ifconfig    # View your network configuration
   wallet      # Check your cryptocurrency and data balance
   ```

2. **Get help:**
   ```bash
   help        # Show all available commands
   ```

3. **Start learning:**
   ```bash
   tutorial    # List available tutorials
   tutorial getting_started  # Start the getting started tutorial
   ```

## Basic Commands

### Filesystem Commands

The game features a virtual filesystem where you can create and manage files:

- `pwd` - Print working directory
- `ls [-l]` - List directory contents (use `-l` for detailed view)
- `cd <directory>` - Change directory (supports `.`, `..`, `~`, absolute paths)
- `cat <filename>` - Display file contents
- `touch <filename>` - Create a new file
- `mkdir <dirname>` - Create a new directory
- `rm <filename>` - Delete file
- `rm -r <folder>` - Delete folder recursively
- `cp <src> <dest>` - Copy files/folders
- `mv <src> <dest>` - Move or rename files/folders
- `edit <filename>` or `vi <filename>` or `nano <filename>` - Edit a file
  - In edit mode: `:save` to save, `:exit` to exit

### System Commands

- `clear` - Clear the screen
- `help` - Show available commands
- `whoami` - Display current username
- `name <newName>` - Change your username
- `info` - Display connection information
- `userinfo` - Display detailed user information (level, experience, resources, wallet)
- `wallet` - Show wallet balance (crypto and data)
- `ascii <text> [flags]` - Convert text to ASCII art
  - Flags:
    - `-h, --help` - Show help message
    - `-a, --animate` - Create animated welcome animation (gradient → ASCII art centered → falls away)
    - `-c, --color <palette>` - Color palette: white, orange, green, purple, blue, red, cyan, yellow, pink
    - `-s, --size <scale>` - Size multiplier: 1-10 (default: 1)
  - Size:
    - Size is an integer multiplier (1-10) that scales the base pattern dimensions
    - Base pattern is 5x7 blocks, so scale 1 = 5x7, scale 2 = 10x14, scale 3 = 15x21, etc.
    - Larger scales = bigger ASCII characters (more blocks per character)
    - Examples: 1 (base), 2 (double), 3 (triple), 5 (5x larger)
  - Color Palettes:
    - `white`, `orange`, `green`, `purple`, `blue`, `red`, `cyan`, `yellow`, `pink`/`magenta`
  - Examples:
    - `ascii HELLO` - Convert "HELLO" to ASCII art
    - `ascii WELCOME -a` - Create animated welcome with "WELCOME" centered
    - `ascii TEST -c green` - Use green color palette
    - `ascii "HELLO WORLD" -s 3` - Triple size (15x21 blocks per character)
    - `ascii TERMINAL -a -c purple -s 2` - Animated with purple palette, double size
    - `ascii -h` - Show detailed help

## Network Exploration

### Scanning

**Scan the internet:**
```bash
scan
```
This discovers servers on the public internet. Servers with shops will be marked with `[SHOP: <type>]`.

**Scan a specific server:**
```bash
scan <targetIP>
```
This reveals:
- Services running on the server
- Vulnerabilities and their security levels
- Server resources (CPU, RAM, bandwidth)
- Security level

**Scan local network (when SSH'd into a server):**
```bash
scan
```
When connected to a server via SSH, `scan` shows servers on that server's local network.

### Connecting to Servers

**SSH into a server:**
```bash
ssh <targetIP>
```
- You must exploit a server before you can SSH into it
- Supports nested SSH (SSH into servers within servers)
- Use `exit` to disconnect and return to the previous server
- Use `exit` at the top level to quit the game

**View current server info:**
```bash
server
```
Shows hardware info when connected to a server via SSH.

## Game Mechanics

### Tool System

Tools are hacking utilities that allow you to exploit servers. Each tool has:
- **Function**: What the tool does
- **Exploits**: Types of vulnerabilities it can exploit (with security levels)
- **Resources**: CPU, bandwidth, and RAM requirements
- **Services**: Which services the tool targets

#### Getting Tools

**Download from the repo server:**
```bash
get repo <toolName>
```
The `repo` server contains all basic tools for free download.

**List your tools:**
```bash
tools
```
Shows all tools you own, their versions, applied patches, and effective exploits.

#### Using Tools

Once you own a tool, you can use it as a command:

**Password Cracking:**
```bash
password_cracker <targetIP>
```
Cracks passwords on a server. Works on servers with password vulnerabilities.

**SSH Exploitation:**
```bash
ssh_exploit <targetIP>
```
Exploits SSH vulnerabilities to gain access.

**User Enumeration:**
```bash
user_enum <targetIP>
```
Enumerates users on a server to gather information.

**Network Sniffing:**
```bash
lan_sniffer <targetIP>
```
Discovers network connections and relationships.

**Password Sniffing:**
```bash
password_sniffer <targetIP>
```
Sniffs and cracks passwords from user roles.

**Rootkit Installation:**
```bash
rootkit <targetIP>
```
Installs a backdoor for persistent access.

**Multi-Exploit Tools:**
```bash
exploit_kit <targetIP>
advanced_exploit_kit <targetIP>
```
These tools can exploit multiple vulnerability types at once.

**Web Exploitation:**
```bash
sql_injector <targetIP>
xss_exploit <targetIP>
```
Target HTTP services specifically.

**Network Analysis:**
```bash
packet_capture <targetIP>
packet_decoder <targetIP>
```
Analyze network traffic. `packet_capture` captures packets, then use `packet_decoder` to decode and analyze them.

**Stealth & Track Covering:**
```bash
log_cleaner <targetIP>
```
Deletes and clears system logs to cover your tracks. Must be used on an exploited server. Essential for stealth missions.

```bash
timestomper <targetIP>
```
Modifies file timestamps to cover tracks and make forensic analysis harder. Must be used on an exploited server.

```bash
audit_disable <targetIP>
```
Disables system auditing and logging to prevent future logs from being created. Must be used on an exploited server. Use this after covering existing tracks.

```bash
backup_destroyer <targetIP>
```
Deletes backups to prevent recovery. Must be used on an exploited server. Use this to ensure data cannot be restored after a heist.

**Data Exfiltration:**
```bash
database_dumper <targetIP>
```
Extracts entire database contents. Requires SQL injection vulnerability (use `sql_injector` first). This tool dumps all database tables and data.

```bash
hash_cracker <targetIP>
```
Advanced hash cracking for MD5, SHA256, bcrypt, and other hash algorithms. Higher success rate than `password_cracker`. Useful for cracking hashed passwords found in databases.

**Intelligence Gathering:**
```bash
log_analyzer <targetIP>
```
Parses and analyzes system logs for intelligence. Must be used on an exploited server. Reveals user activity, failed login attempts, and other valuable information.

**Social Engineering:**
```bash
phishing_kit <targetIP>
```
Creates phishing emails and sites to gather credentials. Must be used on an exploited server. Generates realistic-looking phishing campaigns targeting specific users.

### Exploitation Workflow

1. **Scan for servers:**
   ```bash
   scan
   ```

2. **Scan a specific server:**
   ```bash
   scan <targetIP>
   ```
   Note the services and vulnerabilities.

3. **Download the appropriate tool:**
   ```bash
   get repo <toolName>
   ```

4. **Exploit the server:**
   ```bash
   <toolName> <targetIP>
   ```

5. **SSH into the exploited server:**
   ```bash
   ssh <targetIP>
   ```

6. **Explore the server:**
   - Use filesystem commands to browse
   - Scan the local network for more servers
   - Look for tools to download

**Check exploited servers:**
```bash
exploited
```
Lists all servers you've successfully exploited.

### Cryptocurrency Mining

Mining generates passive cryptocurrency income over time.

**Start mining:**
```bash
crypto_miner <targetIP>
```
- The server must be exploited first
- Server must have sufficient resources (CPU, RAM, bandwidth)
- Mining consumes server resources

**Check active miners:**
```bash
miners
```
Shows all your active mining sessions with resource usage.

**Stop mining:**
```bash
stop_mining <targetIP>
```

**Check your wallet:**
```bash
wallet
```
Shows your cryptocurrency and data balances.

### Shop System

Shops are special servers where you can purchase items. Shops are discovered automatically when you scan servers.

**List discovered shops:**
```bash
shop
```

**Browse a shop:**
```bash
shop <shopID>
```
Shows inventory with prices in crypto and/or data currency.

**Purchase an item:**
```bash
buy <shopID> <itemNumber>
```

**Shop Types:**
- **repo**: Free downloadable resources (tools)
- **tools**: Purchasable tools
- **resources**: CPU/RAM/Bandwidth upgrades
- **mixed**: Combination of the above

**Item Types:**
- **Tools**: New hacking tools
- **Patches**: Tool upgrades (see Patch System)
- **Resources**: Resource upgrades for your user account

### Patch System

Patches upgrade your tools to make them more powerful. Patches can:
- Add new exploit types
- Upgrade existing exploit levels
- Modify resource requirements

**List available patches:**
```bash
patches
```

**View patch details:**
```bash
patch info <patchName>
```

**Apply a patch to a tool:**
```bash
patch <patchName> <toolName>
```

**Check your tool versions:**
```bash
tools
```
Shows tool versions and applied patches.

**Getting Patches:**
- Some patches are free and discoverable
- Others must be purchased from shops
- After purchasing, use `patch <patchName> <toolName>` to apply

### Server Creation

**Create a new server:**
```bash
createServer
```
Creates a new server on the internet with random IP addresses.

**Create a local server:**
```bash
createLocalServer
```
Creates a server on the local network of your current SSH connection. Must be connected to a server first.

## Chat System

The game includes a built-in IRC-style chat system that allows players to communicate in real-time. Chat works seamlessly across both SSH and WebSocket interfaces.

### Features

- **Persistent Rooms**: Chat rooms are stored in the database and survive server restarts
- **Public Rooms**: Anyone can join public rooms (e.g., `#public`)
- **Private Groups**: Create invite-only private rooms
- **Password-Protected Rooms**: Create rooms that require a password to join
- **Tab Navigation**: Switch between multiple rooms using tabs (like IRC clients)
- **Message History**: Last 100 messages per room are persisted
- **Real-Time Messaging**: Messages are broadcast instantly to all users in a room
- **Cross-Platform**: Works identically on both SSH and WebSocket interfaces

### Getting Started

#### Entering Chat Mode

To enter chat mode, simply type:

```bash
chat
```

This will enter full-screen chat mode. You can also use split-screen mode:

```bash
chat --split
```

#### Default Room

When you first enter chat, you'll automatically be joined to the `#public` room (created automatically on first server startup).

### Chat Commands

Once in chat mode, you can use the following commands:

#### Room Management

- `/create <room> [--private|--password <pass>]` - Create a new room
  - Example: `/create myroom` - Creates a public room
  - Example: `/create secret --private` - Creates a private (invite-only) room
  - Example: `/create locked --password secret123` - Creates a password-protected room
- `/join <room> [password]` - Join an existing room
  - Example: `/join #general`
  - Example: `/join locked secret123` - Join with password
- `/leave [room]` - Leave a room (current room if no argument)
  - Example: `/leave #general`
  - Example: `/leave` - Leave current room
- `/rooms` - List all rooms you're currently in
- `/who` - List all users in current room

#### Private Rooms

- `/invite <user> [room]` - Invite a user to a room (current room if no argument)
  - Example: `/invite alice` - Invite to current room
  - Example: `/invite alice secret` - Invite to specific room
  - Note: You must be a member of the room to invite others
  - The invited user receives a notification with instructions to join

#### Navigation

While in chat mode, you can navigate between rooms using:

- **Arrow Keys** (←/→) - Switch between room tabs
- **↑/↓ Arrow Keys** - Navigate command history (like shell)
- **Tab Key** - Autocomplete commands and room names
- **Esc** or **Ctrl+Q** - Exit chat mode

### Room Types

#### Public Rooms

Public rooms can be joined by anyone. Create with `/create`, join with `/join`:

```bash
/create #general           # Create a public room
/join #general             # Join the room
```

#### Private Rooms

Private rooms require an invitation. Only the creator and invited members can join:

```bash
/create secret --private   # Create private room
/invite alice              # Invite alice to current room
# Alice receives: "bob invited you to secret. Use /join secret to enter."
```

#### Password-Protected Rooms

Password-protected rooms require a password to join:

```bash
/create locked --password mypassword   # Create with password
/join locked mypassword                # Join with password
```

### Usage Examples

#### Basic Chatting

```bash
# Enter chat mode
chat

# You're automatically in #public
# Just type your message and press Enter
Hello everyone!

# Create and join another room
/create #general
Hello #general!

# Switch back to #public using arrow keys or tab
# Type another message
How's everyone doing?
```

#### Creating and Managing Rooms

```bash
# Create a private room for your team
/create team-alpha --private

# Invite team members (they'll receive notifications)
/invite bob
/invite charlie

# Create a password-protected room
/create secret-meeting --password secure123

# Share the password with trusted members
# They can join with: /join secret-meeting secure123
```

#### Multi-Room Chatting

```bash
# Create or join multiple rooms
/create #general
/join #public
/join team-alpha

# Use arrow keys or tab to switch between rooms
# Each room maintains its own message history and scroll position
# Use up/down arrows to scroll through message history
```

### Message Format

Messages are displayed in IRC-style format:

```
[15:04:05] <username> Hello everyone!
[15:04:06] <alice> Hey there!
[15:04:07] <bob> What's up?
```

### Chat Tips

- **Navigation**: Use arrow keys (←/→) or Tab to switch rooms
- **Command History**: Use ↑/↓ to cycle through previous commands (like a regular shell)
- **Message History**: Each room keeps the last 100 messages
- **Cross-Interface**: Users on SSH can chat with users on WebSocket - they share the same chat system
- **Room Names**: Room names can start with `#` (like `#public`) or be plain names (like `mygroup`)
- **Invitations**: When invited, you'll receive a notification with the room name and join command
- **Exiting Chat**: Press `Esc` or `Ctrl+Q` to exit chat mode and return to the shell

## Tutorials

The game includes built-in tutorials to help you learn:

**List all tutorials:**
```bash
tutorial
```

**Start a tutorial:**
```bash
tutorial <tutorialID>
```

**Available Tutorials:**
- `getting_started` - Learn the basics of terminal.sh
- `exploitation` - Learn how to exploit servers
- `mining` - Learn cryptocurrency mining
- `advanced_tools` - Learn about advanced exploitation tools
- `story_missions` - Learn about story missions and exclusive rewards

Tutorials have prerequisites, so complete them in order for the best learning experience.

## Story Missions

Story missions are narrative-driven challenges that unlock exclusive tools, patches, and achievements. Complete missions to progress through story arcs and build your hacking arsenal.

### Viewing Missions

**List all available missions:**
```bash
mission
```

This shows all missions organized by story arc, with status indicators:
- ⭕ Not started
- 🔄 In progress
- ✅ Completed

**View mission details:**
```bash
mission <missionID>
```

Shows complete information including:
- Mission description and objectives
- Required tools and level
- Prerequisites (previous missions)
- Rewards (experience, crypto, tools, patches, achievements)

### Starting Missions

**Start a mission:**
```bash
mission start <missionID>
```

Requirements:
- All prerequisite missions must be completed
- You must meet the required level
- You must have the required tools (if any)

**Check your progress:**
```bash
mission status
```

Shows all your started missions with completion percentage.

### Completing Missions

**Complete a mission:**
```bash
mission complete <missionID>
```

After completing all objectives, use this command to finish the mission and receive rewards:
- Experience points
- Cryptocurrency
- **Exclusive tools** (unlocked only through missions)
- **Exclusive patches** (cannot be purchased)
- **Achievements** (displayed in `userinfo`)

### Mission Rewards

Missions unlock exclusive content that cannot be obtained elsewhere:

**Mission-Locked Tools:**
- `log_cleaner` - Delete system logs (from "Cover Your Tracks")
- `timestomper` - Modify file timestamps (from "Cover Your Tracks")
- `database_dumper` - Extract database contents (from "Database Heist")
- `phishing_kit` - Create phishing campaigns (from "Phishing for Answers")
- `hash_cracker` - Advanced hash cracking (from various missions)

**Mission-Exclusive Patches:**
- `stealth_patch_v1` - Reduces log generation
- `sql_injector_stealth` - Stealth mode for SQL injection
- `sql_injector_advanced` - Advanced SQL injection techniques

**Achievements:**
- Unlocked achievements appear in `userinfo`
- Track your progress and accomplishments

### Story Arcs

Missions are organized into story arcs:

**Corporate Espionage Arc:**
1. "The Coffee Shop WiFi" - Hack public WiFi, find credentials
2. "Phishing for Answers" - Create phishing campaign
3. "The Database Heist" - Steal corporate data
4. "Cover Your Tracks" - Cover your tracks before detection

Complete each arc to unlock the next one. Each mission builds on the previous, creating an engaging narrative experience.

### Mission Tips

1. **Check prerequisites first:**
   - Use `mission <id>` to see what's required
   - Complete prerequisite missions before starting new ones

2. **Prepare your tools:**
   - Some missions require specific tools
   - Download tools from `repo` server if needed
   - Mission-locked tools are unlocked by completing missions

3. **Track your progress:**
   - Use `mission status` regularly
   - Check `userinfo` to see unlocked achievements

4. **Plan your strategy:**
   - Missions often require exploiting servers
   - Scan servers first to understand vulnerabilities
   - Use the right tools for each objective

## Tips and Strategies

### Early Game

1. **Start with tutorials:**
   ```bash
   tutorial getting_started
   ```

2. **Scan the internet:**
   ```bash
   scan
   ```

3. **Download basic tools:**
   ```bash
   get repo password_cracker
   get repo ssh_exploit
   ```

4. **Exploit low-security servers first:**
   - Look for servers with security level 1-2
   - Use appropriate tools based on vulnerabilities

5. **Start mining early:**
   - Mining provides passive income
   - Use exploited servers with good resources

### Mid Game

1. **Explore nested networks:**
   - SSH into exploited servers
   - Scan their local networks
   - Find more targets

2. **Upgrade your tools:**
   - Look for patches in shops
   - Apply patches to improve tool effectiveness

3. **Shop for better tools:**
   - Scan servers to discover shops
   - Purchase advanced tools and patches

4. **Build your resource base:**
   - Purchase resource upgrades from shops
   - Higher resources = faster operations

### Advanced Strategies

1. **Tool Specialization:**
   - Focus on tools that match your playstyle
   - Apply multiple patches to favorite tools

2. **Network Mapping:**
   - Use `lan_sniffer` to discover network topology
   - Plan exploitation routes

3. **Resource Management:**
   - Balance mining with active exploitation
   - Monitor server resources

4. **Collaboration:**
   - Use chat to coordinate with other players
   - Share server discoveries
   - Form teams in private chat rooms

### Common Workflows

**Basic Exploitation:**
```bash
scan                    # Find targets
scan <targetIP>         # Analyze target
get repo <tool>         # Get tool
<tool> <targetIP>       # Exploit
ssh <targetIP>          # Access server
```

**Mining Setup:**
```bash
ssh <targetIP>          # Connect to exploited server
server                  # Check resources
exit                    # Return to base
crypto_miner <targetIP> # Start mining
miners                  # Monitor miners
```

**Tool Upgrade:**
```bash
shop                    # Find shops
shop <shopID>           # Browse inventory
buy <shopID> <item>     # Purchase patch
patches                 # List patches
patch <name> <tool>     # Apply patch
tools                   # Verify upgrade
```

## Command Reference

### Filesystem
- `pwd`, `ls`, `cd`, `cat`, `touch`, `mkdir`, `rm`, `cp`, `mv`, `edit`

### System
- `help`, `clear`, `whoami`, `name`, `info`, `userinfo`, `wallet`

### Network
- `scan [targetIP]`, `ifconfig`, `ssh <targetIP>`, `exit`, `server`

### Game
- `get <targetIP> <toolName>`, `tools`, `exploited`
- `createServer`, `createLocalServer`

### Tools (when owned)
- `password_cracker`, `ssh_exploit`, `user_enum`, `lan_sniffer`
- `password_sniffer`, `rootkit`, `exploit_kit`, `advanced_exploit_kit`
- `sql_injector`, `xss_exploit`, `packet_capture`, `packet_decoder`
- `log_cleaner`, `timestomper`, `database_dumper`, `phishing_kit`
- `audit_disable`, `hash_cracker`, `log_analyzer`, `backup_destroyer`

### Mining
- `crypto_miner <targetIP>`, `stop_mining <targetIP>`, `miners`

### Shopping
- `shop [shopID]`, `buy <shopID> <itemNumber>`

### Upgrades
- `patches`, `patch <name> <tool>`, `patch info <name>`

### Learning
- `tutorial [tutorialID]`

### Story Missions
- `mission` - List available missions
- `mission <id>` - View mission details
- `mission start <id>` - Start a mission
- `mission complete <id>` - Complete a mission
- `mission status` - View your mission progress

### Chat
- `chat [--split]`
- Chat commands: `/create`, `/join`, `/leave`, `/rooms`, `/who`, `/invite`

## Troubleshooting

**"Server not found" error:**
- Make sure you've scanned and found the server first
- Check the IP address is correct

**"Tool not owned" error:**
- Download the tool first with `get repo <toolName>`
- Check with `tools` to see what you own

**"Server must be exploited" error:**
- Scan the server first to see vulnerabilities
- Use the appropriate tool to exploit it
- Check `exploited` to verify successful exploitation

**"Insufficient resources" error:**
- Your user account needs more CPU/RAM/bandwidth
- Purchase resource upgrades from shops
- Check your resources with `userinfo`

**Chat not working:**
- Make sure you're in chat mode (type `chat`)
- Check you're in a room (use `/rooms`)
- Try leaving and rejoining the room

**"Prerequisite mission not completed" error:**
- Check mission prerequisites with `mission <missionID>`
- Complete required missions first
- Use `mission status` to see your progress

**"Required level not met" error:**
- Gain experience by using tools and exploiting servers
- Check your level with `userinfo`
- Level up by gaining experience points (100 XP per level)

**"Mission already completed" error:**
- You've already completed this mission
- Check `mission status` to see completed missions
- Move on to the next mission in the arc

## Infinite Gameplay

terminal.sh features a **procedural generation system** that ensures you never run out of content to explore.

### How It Works

When you complete all available static missions or exhaust the available servers, the game automatically generates new content tailored to your skill level:

- **Procedurally Generated Missions**: When you have fewer than 3 available missions, the system automatically creates new missions based on:
  - Your current level
  - Tools you own
  - Your playstyle and progress
  - Available servers to target

- **Procedural Servers**: When the number of available servers drops below 10, new servers are automatically generated with:
  - Difficulty scaled to your level
  - Appropriate vulnerabilities for your tools
  - Local network connections for exploration depth

### Mission Types

Procedurally generated missions include various types:
- **Exploitation**: "Exploit N servers with security level < X"
- **Data Extraction**: "Extract X GB of data from servers"
- **Tool Mastery**: "Use [tool] to exploit N servers"
- **Network Exploration**: "Discover N servers on local networks"
- **Resource Gathering**: "Mine X cryptocurrency"
- **Stealth Operations**: "Cover your tracks on N servers"

### Seamless Experience

The generation happens automatically and transparently - you'll simply see new missions appear in your mission list and new servers when you scan the network. The system ensures:
- Content is always level-appropriate
- Missions require tools you have access to
- Difficulty scales with your progress
- Generated content feels hand-crafted, not repetitive

For technical details, see [PROCEDURAL_GENERATION.md](PROCEDURAL_GENERATION.md).

## Getting Help

- Type `help` in-game for command list
- Use `tutorial` to access built-in tutorials
- Join the `#public` chat room to ask other players
- Check server logs if you're running your own server

Happy hacking!


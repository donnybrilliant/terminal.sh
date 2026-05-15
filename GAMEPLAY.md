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

**Download files from remote servers (when connected):**
- `download <path>` or `dl <path>` - Download a file to `~/Downloads/` on your home computer
  - Must be connected to a server (via `ssh`, `telnet`, `connect`, or `ftp`)
  - Supports absolute paths (`/root/secret.txt`) and relative paths (`secret.txt`)
  - Files are saved to `~/Downloads/` - the Downloads folder is created automatically
  - Examples:
    - `download /root/secret_key.txt` - Download from root's home
    - `download secret.txt` - Download from current directory
    - `download ../etc/config.conf` - Download using relative path

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
- Services running on the server (SSH, Telnet, FTP, HTTP, etc.)
- Vulnerabilities and their security levels
- Server resources (CPU, RAM, bandwidth)
- Security level

**Vulnerability Status:**
- **CLOSED** (red): Vulnerability not yet exploited
- **OPEN** (green): Vulnerability successfully exploited - stays open permanently

**Scan local network (when connected to a server):**
```bash
scan
```
When connected to a server, `scan` shows servers on that server's local network. This enables "server hopping" - exploiting internal servers that are only accessible from within the network.

### Connecting to Servers

To connect to a server, you must first exploit a **shell-granting service** on that server. Different services grant shell access:

| Service | Grants Shell Access | Notes |
|---------|---------------------|-------|
| SSH | Always | Primary shell access method |
| Telnet | Always | Legacy protocol, often weaker security |
| FTP | Only with RCE | Requires Remote Code Execution vulnerability |
| HTTP | Never | Data extraction only (SQL injection, XSS) |
| MySQL | Never | Data extraction only |

#### Connection Commands

**Connect via any exploited service (auto-detect):**
```bash
connect <targetIP>
```
Automatically uses the first shell-granting service you've exploited.

**Connect via specific service:**
```bash
ssh <targetIP>      # Connect via SSH (requires SSH to be exploited)
telnet <targetIP>   # Connect via Telnet (requires Telnet to be exploited)
ftp <targetIP>      # Connect via FTP (requires FTP with RCE exploit)
```

**Examples:**
```bash
# After exploiting SSH on "test" server
ssh test            # Connects via SSH

# After exploiting Telnet on "test" server  
telnet test         # Connects via Telnet

# Auto-detect - uses whichever service you've exploited
connect test        # Uses SSH, Telnet, or FTP (whichever is available)
```

**Connection Notes:**
- You must exploit a shell-granting service before you can connect
- Supports nested connections (connect to servers within servers for "server hopping")
- Use `exit` to disconnect and return to the previous server
- Use `exit` at the top level to quit the game
- The connection message shows which service type was used

**View current server info:**
```bash
server
```
Shows hardware info when connected to a server.

## Game Mechanics

### Tool System

Tools are hacking utilities that allow you to exploit servers. Each tool has:
- **Function**: What the tool does
- **Exploits**: Types of vulnerabilities it can exploit (with security levels)
- **Resources**: CPU, bandwidth, and RAM requirements
- **Services**: Which services the tool targets

#### Getting Tools

**Unlock via missions and shops first:**
- Early tools are granted by story missions.
- Advanced tools unlock later via shops or additional repos.
- Tools hosted on servers are only visible after you gain access.

**Download from the repo server (when unlocked):**
```bash
get repo <toolName>
```
The `repo` server contains basic tools once your progression unlocks access.

**List your tools:**
```bash
tools
```
Shows all tools you own, their versions, applied patches, and effective exploits.

#### Using Tools

Once you own a tool, you can use it as a command. The game features a **realistic exploitation flow**:

### Credential-Based Access (Password Cracking)

This is the most common path to gaining access:

**Step 1: Enumerate Users**
```bash
user_enum <targetIP>
```
Discovers usernames on the server. This information helps `password_cracker` crack more accounts.

**Step 2: Crack Passwords**
```bash
password_cracker <targetIP>
```
Cracks passwords for discovered users. Works on services with `password_cracking` vulnerabilities:
- SSH (port 22)
- Telnet (port 23)
- FTP (port 21)

If you ran `user_enum` first, you'll crack all discovered users. Otherwise, it tries common usernames.

**Step 3: Connect with Credentials**
```bash
ssh <targetIP>      # Connect via SSH with cracked credentials
telnet <targetIP>   # Connect via Telnet
connect <targetIP>  # Auto-detect service
```

### Direct Access (RCE Exploits)

For servers with Remote Code Execution vulnerabilities, you can gain direct shell access without credentials:

**SSH Exploitation:**
```bash
ssh_exploit <targetIP>
```
Exploits RCE/buffer overflow vulnerabilities on SSH to install a **backdoor**. This gives you immediate root access without needing passwords.

### Managing Access

**View all your access:**
```bash
exploited          # Show all server access (credentials + backdoors)
credentials        # List all discovered username/password pairs
backdoors          # List all installed backdoors
```

### Reconnaissance Tools

**User Enumeration:**
```bash
user_enum <targetIP>
```
Discovers usernames on a server. Run this before `password_cracker` for better results.

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

There are two paths to gaining server access:

#### Path A: Credential Cracking (Most Common)

1. **Scan for servers:**
   ```bash
   scan
   ```

2. **Scan a specific server:**
   ```bash
   scan <targetIP>
   ```
   Note the services and `password_cracking` vulnerabilities.

3. **Download tools:**
   ```bash
   get repo user_enum
   get repo password_cracker
   ```

4. **Enumerate users (optional but recommended):**
   ```bash
   user_enum <targetIP>
   ```
   Discovers usernames - `password_cracker` will crack all discovered users.

5. **Crack passwords:**
   ```bash
   password_cracker <targetIP>
   ```
   Outputs cracked username:password pairs.

6. **Connect with credentials:**
   ```bash
   ssh <targetIP>      # Uses cracked credentials
   connect <targetIP>  # Auto-detect
   ```

#### Path B: RCE Exploit (Direct Access)

1. **Find server with RCE vulnerability:**
   ```bash
   scan <targetIP>
   ```
   Look for `remote_code_execution` or `buffer_overflow` vulnerabilities.

2. **Download RCE tool:**
   ```bash
   get repo ssh_exploit
   ```

3. **Exploit and install backdoor:**
   ```bash
   ssh_exploit <targetIP>
   ```
   Installs a backdoor with root access - no credentials needed!

4. **Connect via backdoor:**
   ```bash
   ssh <targetIP>
   ```

#### After Gaining Access

**Explore the server:**
- Use filesystem commands (`ls`, `cd`, `cat`)
- Check `/var/log/` for useful information
- Scan the local network for internal servers
- Look for tools or sensitive data

**Manage your access:**
```bash
exploited      # Show all access (credentials + backdoors)
credentials    # List cracked passwords
backdoors      # List installed backdoors
```

### Role-Based Access System

Servers have multiple user accounts (roles) with different privilege levels. Your access level determines what you can do on a server.

#### Role Types

| Role Type | Symbol | Home Directory | Can Access |
|-----------|--------|----------------|------------|
| **root** | `#` | `/root` | Everything - full system access |
| **admin** | `$` | `/home/<username>` | Most files, can sudo |
| **user** | `$` | `/home/<username>` | Own home, `/tmp`, public files |
| **guest** | `$` | `/home/<username>` | Read-only, very limited |

#### Prompt Indicators

The shell prompt shows your current user and privilege level:

```bash
root@192.168.1.100:/root#      # Root user (# prompt)
admin@192.168.1.100:/home/admin$   # Admin user ($ prompt)
user@192.168.1.100:/home/user$     # Regular user ($ prompt)
```

#### File Access Permissions

Different roles have different filesystem permissions:

| Path | root | admin | user | guest |
|------|------|-------|------|-------|
| `/root/*` | ✅ | ❌ | ❌ | ❌ |
| `/etc/shadow` | ✅ | ❌ | ❌ | ❌ |
| `/etc/sudoers` | ✅ | ❌ | ❌ | ❌ |
| `/home/<user>/*` | ✅ | ✅ (own) | ✅ (own) | ❌ |
| `/var/log/*` | ✅ | ✅ (read) | ✅ (read) | ❌ |
| `/tmp/*` | ✅ | ✅ | ✅ | ✅ (read) |

#### How Role is Determined

Your role depends on how you gained access:

- **Credential cracking** → Connects as the cracked user (user, admin, etc.)
- **RCE backdoor** → Usually grants root access
- **Privilege escalation** → Upgrades current role to root

### Privilege Escalation

If you connect as a low-privilege user (not root), you can attempt to escalate to root using local vulnerabilities.

#### Workflow

1. **Connect with non-root credentials:**
   ```bash
   ssh <targetIP>   # Connects as 'user' if you cracked user's password
   ```

2. **Scan for privilege escalation vectors:**
   ```bash
   privesc_scanner
   ```
   Shows local vulnerabilities like sudo misconfigurations, SUID binaries, kernel vulns.

3. **Exploit a vulnerability:**
   ```bash
   sudo_exploit     # Exploit sudo misconfiguration
   suid_finder      # Find and exploit SUID binaries
   kernel_exploit   # Exploit kernel vulnerability
   ```

4. **Reconnect with root:**
   ```bash
   exit
   ssh <targetIP>   # Now connects as root!
   ```

#### Privilege Escalation Tools

| Tool | What It Exploits | Difficulty |
|------|------------------|------------|
| `privesc_scanner` | Scans for all local vulns | - |
| `sudo_exploit` | Sudo misconfigurations | Medium |
| `suid_finder` | SUID binary vulnerabilities | Medium |
| `kernel_exploit` | Kernel CVEs | Hard |

#### Example: Privilege Escalation

```bash
# 1. You cracked 'user' password and connected
user@192.168.1.100:/home/user$ 

# 2. Try to read root's secrets - permission denied!
user@192.168.1.100:/home/user$ cd /root
Permission denied

# 3. Scan for privilege escalation
user@192.168.1.100:/home/user$ privesc_scanner
Found 2 potential privilege escalation vectors:
  1. SUDO Misconfiguration Level 8 → root
     User can run vim as root without password
     Target: /usr/bin/vim

# 4. Exploit it!
user@192.168.1.100:/home/user$ sudo_exploit
✅ PRIVILEGE ESCALATION SUCCESSFUL!
Now running as: root

# 5. Reconnect to use root
user@192.168.1.100:/home/user$ exit
$ ssh 192.168.1.100
root@192.168.1.100:/root# 

# 6. Now we can access everything!
root@192.168.1.100:/root# cat secret_key.txt
MASTER_KEY=sk_live_9x8y7z6w5v4u3t2s1r0q
```

#### Why Privilege Escalation Matters

- **Access sensitive files** like `/root/*`, `/etc/shadow`
- **Find more credentials** stored in root's home
- **Install persistent backdoors**
- **Complete missions** that require root access
- **Higher rewards** from fully compromising servers

### Server Logs

Servers maintain dynamic logs that record activity. When you read log files on a server, you'll see both pre-existing (seeded) logs and real-time activity logs from your actions and other players.

**Key log files:**
- `/var/log/auth.log` - Authentication events (connections, exploits, login attempts)
- `/var/log/system.log` - System events (commands, file access, scans)

**Log entries include:**
- **Connections**: When users connect/disconnect via SSH, Telnet, FTP
- **Exploit attempts**: Both successful and failed exploitation attempts
- **Scans**: Port scans detected from specific IPs
- **Commands**: Commands executed on the server

**IP Tracking:**
Logs show the **source IP** of each action. When server hopping, the source IP reflects your last hop:
- Direct connection from your machine → your IP is logged
- Connection from Server A to Server B → Server A's IP is logged on Server B

This is important for stealth - use `log_cleaner` to cover your tracks!

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
This shows:
- **Accessible shops** - Shops you can browse and purchase from
- **Locked shops** - Shops that require mission completion or higher level

**Browse a shop:**
```bash
shop <shopID>
```
Shows inventory with prices in crypto and/or data currency.

**Purchase an item:**
```bash
buy <shopID> <itemNumber>
```

**Shop Unlocking:**
Some shops require completing missions or reaching certain levels:
- **Resource Boost Shop** - Unlocks after completing "The Coffee Shop WiFi" (corp_espionage_01)
- **Elite Tools Shop** - Unlocks after completing "Secure Shell Access" (corp_espionage_03) and reaching Level 3

Locked shops will show requirements in the shop list.

**Shop Types:**
- **repo**: Free downloadable resources (tools)
- **tools**: Purchasable upgrade tokens
- **resources**: CPU/RAM/Bandwidth upgrades
- **mixed**: Combination of the above

**Item Types:**
- **Upgrade Tokens**: Free tool upgrades (exploit boost, CPU/RAM/BW optimization)
- **Resources**: Resource upgrades for your user account (CPU, RAM, Bandwidth boosts)

Note: Tools are primarily obtained through mission rewards, not shops.

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
- `privilege_escalation` - Learn how to escalate privileges
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

### Story vs Board Missions

**Story missions** start automatically on triggers (e.g., `cat README.txt` starts the home recovery mission). Objectives complete automatically as you play—no need to type "mission complete".

**Mission board** missions require you to accept them:
```bash
mission start <missionID>
```

**Abandon a mission:**
```bash
mission stop <missionID>
```

**Check your progress:**
```bash
mission status
```

This shows:
- Story arc completion percentages
- Endless mode status (if unlocked)
- Active missions with real-time objective progress
- Remaining objectives for each mission

### Mission Rewards

Missions unlock exclusive content that cannot be obtained elsewhere:

**Mission-Locked Tools:**
- `packet_capture`, `password_sniffer`, `password_cracker` - Early access tools from WiFi/Legacy missions
- `ssh_exploit`, `user_enum` - Secure Shell access mission
- `privesc_scanner`, `sudo_exploit`, `kernel_exploit`, `suid_finder` - Privilege escalation mission
- `phishing_kit`, `database_dumper`, `hash_cracker` - Corporate espionage missions
- `log_cleaner`, `timestomper`, `audit_disable` - Cover Your Tracks mission
- `lan_sniffer`, `packet_decoder`, `log_analyzer`, `rootkit` - Field Operations arc
- `exploit_kit`, `advanced_exploit_kit`, `xss_exploit` - Scale exploitation mission
- `crypto_miner`, `backup_destroyer` - Late Field Operations missions

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
1. "The Coffee Shop WiFi" - Capture traffic and extract credentials
2. "Legacy Access" - Telnet/FTP password access
3. "Secure Shell Access" - SSH enumeration and exploitation
4. "Privilege Escalation" - Escalate to root locally
5. "Phishing for Answers" - Launch phishing campaigns
6. "The Database Heist" - SQL injection + data extraction
7. "Cover Your Tracks" - Clean logs and disable auditing

**Field Operations Arc:**
1. "Network Recon" - LAN sniffing and log analysis
2. "Persistent Access" - Rootkit installation
3. "Exploit at Scale" - Multi-exploit tooling
4. "Resource Hijack" - Crypto mining
5. "Backup Erasure" - Destroy recovery data

Complete each arc to unlock the next one. Each mission builds on the previous, creating an engaging narrative experience.

### Endless Mode

After completing at least one story arc, **Endless Mode** is unlocked! This provides infinite gameplay through procedurally generated content.

**What Endless Mode offers:**
- **Procedural Missions** - New missions generated based on your level and capabilities
- **Procedural Servers** - Fresh servers to exploit, scaled to your tier
- **10-Tier Difficulty System** - Content scales from Tier 1 (beginner) to Tier 10 (expert)
- **Server Recycling** - Depleted servers are replaced with new challenges

**Check your Endless Mode status:**
```bash
mission status
```

This shows:
- Story arcs completed
- Procedural missions completed
- Highest tier reached (1-10)
- Total servers exploited

**Tier System:**
| Tier | Player Level | Security Range | Features |
|------|--------------|----------------|----------|
| 1 | 1 | 1-10 | Basic password cracking |
| 2 | 2 | 10-20 | SSH, Telnet, FTP |
| 3 | 3 | 20-30 | HTTP, SQL injection |
| 4 | 4 | 25-35 | XSS, advanced exploits |
| 5 | 5 | 30-40 | Complex vulnerabilities |
| 6 | 6 | 35-50 | **Privilege escalation unlocked** |
| 7 | 7 | 45-60 | Buffer overflow, rootkits |
| 8 | 8 | 55-70 | Advanced local exploits |
| 9 | 9 | 65-85 | Multiple local vulnerabilities |
| 10 | 10+ | 80-100 | Maximum difficulty |

Local privilege escalation vulnerabilities (sudo misconfig, SUID binaries, kernel exploits) only appear at Tier 6 and above.

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
   mission start corp_espionage_01
   ```

4. **Exploit low-security servers first:**
   - Look for servers with security level 1-2
   - Use appropriate tools based on vulnerabilities

5. **Start mining early:**
   - Mining provides passive income
   - Use exploited servers with good resources

### Mid Game

1. **Explore nested networks (Server Hopping):**
   - Connect to exploited servers using `connect`, `ssh`, `telnet`, or `ftp`
   - Scan their local networks to find internal servers
   - Exploit and connect to internal servers for deeper access
   - Note: Internal servers may have weaker security but contain valuable data

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
- `pwd`, `ls`, `cd`, `cat`, `touch`, `mkdir`, `rm`, `cp`, `mv`, `edit`, `download`/`dl`

### System
- `help`, `clear`, `whoami`, `name`, `info`, `userinfo`, `wallet`

### Network
- `scan [targetIP]`, `ifconfig`, `server`, `exit`
- `connect <targetIP>` - Auto-detect and connect via any exploited service
- `ssh <targetIP>` - Connect via SSH
- `telnet <targetIP>` - Connect via Telnet
- `ftp <targetIP>` - Connect via FTP (requires RCE exploit)

### Game
- `get <targetIP> <toolName>` - Download tool from server
- `tools` - List owned tools
- `exploited` - Show all server access (credentials + backdoors)
- `credentials` - List discovered username/password pairs
- `backdoors` - List installed backdoors
- `createServer`, `createLocalServer` - Create servers

### Tools (when owned)
- **Reconnaissance:** `user_enum`, `lan_sniffer`, `packet_capture`, `packet_decoder`, `log_analyzer`
- **Credential Attacks:** `password_cracker`, `password_sniffer`, `hash_cracker`, `phishing_kit`
- **Exploits:** `ssh_exploit`, `exploit_kit`, `advanced_exploit_kit`, `sql_injector`, `xss_exploit`
- **Privilege Escalation:** `privesc_scanner`, `sudo_exploit`, `kernel_exploit`, `suid_finder`
- **Post-Exploitation:** `rootkit`, `log_cleaner`, `timestomper`, `audit_disable`, `database_dumper`, `backup_destroyer`

### Mining
- `crypto_miner <targetIP>`, `stop_mining <targetIP>`, `miners`

### Shopping
- `shop [shopID]`, `buy <shopID> <itemNumber>`

### Upgrades
- `patches`, `patch <name> <tool>`, `patch info <name>`

### Learning
- `tutorial [tutorialID]`

### Story Missions
- `mission` - List missions (story + board)
- `mission <id>` - View mission details
- `mission start <id>` - Accept a board mission
- `mission stop <id>` - Abandon a mission
- `mission status` - View your progress

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
- Complete required missions first (they auto-complete when objectives are done)
- Use `mission status` to see your progress

**"Required level not met" error:**
- Gain experience by using tools and exploiting servers
- Check your level with `userinfo`
- Level up by gaining experience points (100 XP per level)

**"Mission already completed" error:**
- You've already completed this mission
- Check `mission status` to see completed missions
- Move on to the next mission in the arc

**"Cannot abandon story mission" error:**
- Story missions (trigger-based) cannot be stopped
- Use `mission stop` only for mission board missions

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
  - Variety of services (SSH, Telnet, FTP, HTTP) with different access methods

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


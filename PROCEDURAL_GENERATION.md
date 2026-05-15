# Procedural Generation System

## Overview

The Procedural Generation System ensures infinite gameplay by automatically creating missions and servers when static content is exhausted. This system analyzes player state and generates personalized, level-appropriate content.

## Architecture

### Components

1. **MissionGenerator** (`services/mission_generator.go`)
   - Analyzes player state (level, tools, progress)
   - Generates missions based on player capabilities
   - Creates objectives and calculates rewards
   - Stores generated missions in database

2. **ServerGenerator** (`services/server_generator.go`)
   - Detects when servers are exhausted
   - Generates new servers with appropriate difficulty
   - Creates local network connections
   - Tracks procedural servers

3. **Database Models**
   - `GeneratedMission` - Tracks procedurally generated missions per user
   - `ProceduralServer` - Tracks procedurally generated servers

## How It Works

### Mission Generation

**Trigger**: When `MissionService.GetAvailableMissions()` returns fewer than 3 missions

**Process**:
1. Analyze player state (level, owned tools, exploited servers, completed missions)
2. Select appropriate mission type based on player capabilities
3. Generate objectives scaled to player level
4. Calculate rewards based on difficulty
5. Store mission in database with tracking record

**Mission Types**:
- `exploitation` - Always available, scales with level
- `data_extraction` - Requires level 3+ and available servers
- `tool_mastery` - Requires owned tools
- `network_exploration` - Requires level 2+ and available servers
- `resource_gathering` - Requires level 1+
- `stealth` - Requires level 5+

### Server Generation

**Trigger**: When `NetworkService.ScanInternet()` finds fewer than 10 servers

**Process**:
1. Check current server count
2. Calculate how many servers needed (minimum 5, up to 15 per generation)
3. For each server:
   - Calculate security level: `baseLevel + (playerLevel * 2) + random(-10, +10)`
   - Generate services (SSH, HTTP, or both) based on mission needs
   - Create vulnerabilities appropriate for level
   - Optionally create 1-3 local servers (30% chance)
4. Track generated servers in database

### Difficulty Scaling - 10-Tier System

The game uses a 10-tier difficulty system that maps to player levels:

| Tier | Player Level | Security Range | Vulnerabilities | Features |
|------|--------------|----------------|-----------------|----------|
| 1 | 1 | 1-10 | password_cracking 1-5 | SSH, Telnet |
| 2 | 2 | 10-20 | password_cracking 5-10 | + FTP |
| 3 | 3 | 20-30 | RCE 10-15, SQL 10-15 | + HTTP |
| 4 | 4 | 25-35 | RCE 15-20, XSS 10-15 | Advanced web |
| 5 | 5 | 30-40 | RCE 20-25, SQL 15-20 | Complex chains |
| 6 | 6 | 35-50 | All types + **local vulns** | Priv esc unlocked |
| 7 | 7 | 45-60 | + buffer_overflow | Rootkit targets |
| 8 | 8 | 55-70 | Advanced exploits | Multiple local vulns |
| 9 | 9 | 65-85 | High-level exploits | 1-3 local vulns |
| 10 | 10+ | 80-100 | Maximum difficulty | All features |

**Local Privilege Escalation** (Tier 6+):
- `sudo_misconfiguration` - Misconfigured sudo rules
- `suid_binary` - SUID binaries with unsafe input handling
- `kernel_exploit` - Outdated kernel vulnerabilities (CVE-based)

**Missions**:
- Base XP: `100 * (1 + level * 0.1)`
- Base Crypto: `20.0 * (1 + level * 0.15)`
- Objective count: `min + (level / 3)`, capped at max
- Security level targets: Based on tier ranges

**Servers**:
- Security level: Calculated from tier range
- Resources scale with level
- Local servers: `parentLevel - (10 to 30)`
- Local network depth increases with tier (1-3: 0-1, 4-6: 1-2, 7+: 1-3)

## Configuration

Constants defined in `config/config.go`:
- `MinServersOnline = 10` - Minimum servers to keep available
- `MissionsPerUser = 5` - How many missions to generate per user
- `GenerationInterval = 3600` - Check interval in seconds (1 hour)
- `MaxGeneratedServers = 1000` - Limit on procedural servers

## Integration Points

### MissionService Integration

```go
// In GetAvailableMissions()
if len(available) < 3 && s.missionGenerator != nil {
    // Generate missions automatically
    generatedMission, err := s.missionGenerator.GenerateMission(userID)
    // Add to available missions
}
```

### NetworkService Integration

```go
// In ScanInternet()
if n.serverGenerator != nil && len(servers) < config.MinServersOnline {
    // Generate servers automatically
    n.serverGenerator.CheckAndGenerateServers(userID)
}
```

## Database Schema

### GeneratedMission
- `ID` - UUID primary key
- `UserID` - User who owns this mission
- `MissionID` - Generated mission ID
- `GeneratedAt` - When mission was created
- `Difficulty` - Calculated difficulty level
- `ServerIP` - Target server (if mission-specific)
- `MissionData` - JSON of complete mission definition

### ProceduralServer
- `ID` - UUID primary key
- `ServerID` - Reference to Server.ID
- `GeneratedFor` - User ID or mission ID
- `GeneratedAt` - When server was created
- `Reason` - "exhaustion", "mission", or "proactive"

## User Experience

- **Seamless**: Players don't notice when content is generated
- **Natural**: Missions appear in mission list, servers appear when scanning
- **Variety**: Generated content feels hand-crafted, not repetitive
- **Progressive**: Difficulty scales appropriately with player level

## Server Lifecycle Management

Procedural servers have a lifecycle to prevent infinite accumulation:

### Depletion Detection
- Servers are marked as "depleted" when 90%+ of resources have been extracted
- Depleted servers no longer provide meaningful loot

### Cleanup System
- `CleanupDepletedServers()` - Removes fully exploited procedural servers
- `EnforceServerLimit()` - Keeps procedural servers under 100 maximum
- Oldest depleted servers are removed first
- Non-depleted servers are only removed if over limit

### Server Recycling
- Depleted servers can be recycled instead of deleted
- `RecycleServer()` regenerates content scaled to current player level
- Same IP, fresh vulnerabilities and resources

## Action Tracking System

A centralized system tracks player actions for mission validation:

### Tracked Actions
- `tool_use` - Tool execution with target server
- `server_exploit` - Successful exploitation
- `privilege_escalate` - Privilege escalation
- `credential_crack` - Credential cracking
- `data_extract` - Data extraction
- `backdoor_install` - Backdoor installation

### Mission Validation
- Actions are linked to active missions
- `HasCompletedObjective()` validates each mission objective
- `mission complete` verifies all objectives before granting rewards
- Players cannot complete missions without actually doing the work

### Integration
Tool handlers call tracking methods:
```go
h.trackToolUse("password_cracker", targetIP, "ssh")
h.trackServerExploit("ssh_exploit", serverPath, "ssh")
```

## Post-Story Endless Mode

After completing story arcs, endless mode provides infinite content:

### Unlock Conditions
- Complete at least one story arc
- Endless mode status shown in `mission status`

### Endless Mode Features
- Procedural missions generated indefinitely
- Servers regenerated at player's tier level
- Progress tracking (procedural missions completed, highest tier)

### Story Arc Progress
- `GetStoryArcProgress()` - Returns completion % for each arc
- `IsStoryComplete()` - Checks if all arcs completed
- `GetEndlessModeStatus()` - Returns endless mode statistics

## Future Enhancements

1. **Multiplayer Missions** - Missions requiring multiple players
2. **Seasonal Events** - Special missions during events
3. **Player-Created Content** - Players can create missions for others
4. **Machine Learning** - Learn from player behavior to improve generation
5. **Procedural Stories** - Generate story arcs, not just missions
6. **Leaderboards** - Track endless mode progress across players
7. **Prestige System** - Reset progress for bonuses after tier 10

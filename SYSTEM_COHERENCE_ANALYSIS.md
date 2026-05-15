# System Coherence Analysis

## Overview

This document analyzes how all game systems (tools, shops, servers, missions, tutorials) fit together and identifies any inconsistencies or gaps.

## Tool Distribution Strategy

### ✅ Repo Server (Free Tools)
**Location**: `repo` server (IP: "repo")
**Access**: `get repo <toolName>` (free download)
**Purpose**: Basic tools for new players

**Tools Available in Repo** (from `seedServersFromJSON`):
- ALL tools from `tools.json` are automatically added to repo
- This includes: password_cracker, ssh_exploit, user_enum, lan_sniffer, rootkit, exploit_kit, sql_injector, xss_exploit, packet_capture, packet_decoder, password_sniffer, crypto_miner
- **Also includes story-critical tools**: log_cleaner, timestomper, database_dumper, phishing_kit, audit_disable, hash_cracker, log_analyzer, backup_destroyer

**⚠️ ISSUE FOUND**: Story-critical tools are in repo, but missions unlock them as rewards. This breaks the exclusivity!

### ✅ Shops (Purchasable Tools)
**Location**: Various shop servers (e.g., 1.1.1.1, 2.2.2.2)
**Access**: `buy <shopID> <itemNumber>` (costs crypto/data)
**Purpose**: Premium tools and upgrades

**Current Shop Inventory** (from `shops.json`):
- **Elite Tools Shop** (1.1.1.1):
  - `advanced_exploit_kit` (1000 crypto) ✅
  - `pass_patch_v2` (500 data) ✅
  - `ssh_patch_v2` (750 data) ✅
  - `cpu_boost` (500 crypto) ✅
- **Resource Boost Shop** (2.2.2.2):
  - `bandwidth_boost` (300 crypto) ✅
  - `ram_boost` (400 crypto) ✅
  - `full_boost` (2000 crypto, 1000 data) ✅

**✅ GOOD**: Shops only sell premium items, not basic tools.

### ✅ Missions (Exclusive Tools)
**Location**: Unlocked through story missions
**Access**: Complete mission → receive tool as reward
**Purpose**: Story-driven tool unlocks

**Mission-Locked Tools** (from mission rewards):
- `phishing_kit` - Mission 2 reward ✅
- `database_dumper` - Mission 3 reward ✅
- `log_cleaner` - Mission 4 reward ✅
- `timestomper` - Mission 4 reward ✅

**⚠️ CRITICAL ISSUE**: These tools are ALSO in the repo server! This breaks mission exclusivity.

## Mission Requirements Analysis

### Mission 1: "The Coffee Shop WiFi"
- **Required Tools**: `packet_capture`, `password_sniffer`
- **Status**: ✅ Both available in repo (good for new players)
- **Rewards**: Achievement only (no tools)
- **Issue**: None - appropriate for first mission

### Mission 2: "Phishing for Answers"
- **Required Tools**: None
- **Status**: ✅ Good - mission teaches tool usage
- **Rewards**: `phishing_kit` (unlocks tool)
- **Issue**: ⚠️ `phishing_kit` is also in repo - breaks exclusivity

### Mission 3: "The Database Heist"
- **Required Tools**: `sql_injector`
- **Status**: ✅ Available in repo (good)
- **Rewards**: `database_dumper` (unlocks tool)
- **Issue**: ⚠️ `database_dumper` is also in repo - breaks exclusivity

### Mission 4: "Cover Your Tracks"
- **Required Tools**: `sql_injector`, `database_dumper`
- **Status**: ✅ `sql_injector` in repo, `database_dumper` from Mission 3
- **Rewards**: `log_cleaner`, `timestomper` (unlock tools)
- **Issue**: ⚠️ Both tools are also in repo - breaks exclusivity

## Server Generation Analysis

### Repo Server
- **Security Level**: 200 (unexploitable) ✅
- **Tools**: All tools from `tools.json` ✅
- **Services**: SSH only, not vulnerable ✅
- **Status**: Correctly configured

### Shop Servers
- **Security Level**: 30-50 (exploitable) ✅
- **Tools**: None (shops sell items, not tools) ✅
- **Services**: SSH with vulnerabilities ✅
- **Status**: Correctly configured - players can exploit shop servers but shops still work

### Test Server
- **Security Level**: 15 (very exploitable) ✅
- **Tools**: `password_cracker`, `ssh_exploit` ✅
- **Purpose**: Tutorial/testing server ✅
- **Status**: Correctly configured

## Tutorial Alignment

### ✅ Tutorials Match Available Tools
- `getting_started` - Basic commands ✅
- `exploitation` - Basic tools (password_cracker, ssh_exploit) ✅
- `mining` - crypto_miner ✅
- `advanced_tools` - user_enum, lan_sniffer, exploit_kit, sql_injector, packet_capture, password_sniffer, rootkit, xss_exploit, packet_decoder ✅
- `cover_your_tracks` - log_cleaner, timestomper, audit_disable ✅
- `data_heist` - database_dumper, hash_cracker ✅

**Status**: All tutorials reference tools that exist in the game ✅

## Critical Issues Found

### 🔴 Issue #1: Mission Exclusivity Broken

**Problem**: Mission-locked tools (`phishing_kit`, `database_dumper`, `log_cleaner`, `timestomper`) are available in the repo server, making missions less meaningful.

**Impact**: 
- Players can bypass missions to get tools
- Mission rewards feel less valuable
- Story progression is optional rather than required

**Solution**: Remove mission-locked tools from repo server. The repo should only contain:
- Basic tools (password_cracker, ssh_exploit, user_enum, lan_sniffer)
- Common tools (exploit_kit, sql_injector, xss_exploit, packet_capture, packet_decoder)
- Mining tool (crypto_miner)
- **NOT** mission-exclusive tools

### 🟡 Issue #2: Missing Tools in Shops

**Problem**: Shops only sell `advanced_exploit_kit` and patches. According to brainstorming plan, shops should also sell:
- `rootkit` (premium tool)
- `password_sniffer` (premium tool)
- Other advanced tools

**Impact**: Limited shop inventory makes shops less useful

**Solution**: Add more premium tools to shops (but NOT mission-exclusive tools)

### 🟡 Issue #3: Mission Hints Reference Tools Not Yet Owned

**Problem**: Mission hints tell players to use tools they don't have yet (e.g., Mission 2 hint says to use `phishing_kit` before they've unlocked it).

**Impact**: Confusing for players - they see hints but can't complete objectives

**Solution**: Hints should explain that the tool will be unlocked as a reward, or missions should grant tools before requiring their use.

### 🟢 Issue #4: Patch Distribution

**Current State**:
- Some patches in shops (`pass_patch_v2`, `ssh_patch_v2`)
- Some patches in mission rewards (`sql_injector_stealth`, `stealth_patch_v1`)
- Some patches in server filesystems (discoverable)

**Status**: ✅ This is actually good - patches have multiple sources, creating variety

## Recommended Fixes

### Priority 1: Fix Mission Exclusivity

**Action**: Modify repo server tool list to exclude mission-locked tools.

**Tools to REMOVE from repo**:
- `phishing_kit` (Mission 2 reward)
- `database_dumper` (Mission 3 reward)
- `log_cleaner` (Mission 4 reward)
- `timestomper` (Mission 4 reward)
- `audit_disable` (should be mission-locked)
- `hash_cracker` (should be mission-locked)
- `log_analyzer` (should be mission-locked)
- `backup_destroyer` (should be mission-locked)

**Tools to KEEP in repo**:
- `password_cracker`
- `ssh_exploit`
- `user_enum`
- `lan_sniffer`
- `password_sniffer` (or move to shop?)
- `rootkit` (or move to shop?)
- `exploit_kit`
- `advanced_exploit_kit` (or keep in shop only?)
- `sql_injector`
- `xss_exploit`
- `packet_capture`
- `packet_decoder`
- `crypto_miner`

### Priority 2: Expand Shop Inventory

**Action**: Add premium tools to shops (but not mission-exclusive ones).

**Suggested Shop Additions**:
- `rootkit` (500 crypto) - Premium persistence tool
- `password_sniffer` (300 crypto) - Better than password_cracker
- `hash_cracker` (800 crypto) - If not mission-locked
- More patches and resource upgrades

### Priority 3: Fix Mission Flow

**Action**: Ensure mission hints don't reference tools players don't have.

**Options**:
1. Grant tools at mission START (not completion)
2. Update hints to explain tools will be unlocked
3. Make mission objectives clearer about tool availability

## Tool Distribution Strategy (Recommended)

### Free (Repo Server)
- Basic exploitation: password_cracker, ssh_exploit
- Information gathering: user_enum, lan_sniffer
- Basic web: sql_injector, xss_exploit
- Network: packet_capture, packet_decoder
- Mining: crypto_miner
- Basic multi-exploit: exploit_kit

### Premium (Shops)
- `advanced_exploit_kit` (already in shop)
- `rootkit` (add to shop)
- `password_sniffer` (add to shop)
- Patches and resource upgrades

### Mission-Exclusive (Missions Only)
- `phishing_kit` (Mission 2)
- `database_dumper` (Mission 3)
- `log_cleaner` (Mission 4)
- `timestomper` (Mission 4)
- `audit_disable` (future mission)
- `hash_cracker` (future mission)
- `log_analyzer` (future mission)
- `backup_destroyer` (future mission)

## Server Generation Consistency

### ✅ Good Practices
- Repo server is secure (level 200, not exploitable)
- Shop servers are exploitable but shops still work
- Test server is easy to exploit for tutorials
- Random servers have appropriate security levels

### 🟡 Potential Improvements
- Add mission-specific servers (e.g., "coffee shop WiFi server", "audit server")
- Ensure servers have appropriate services for mission objectives
- Add servers with specific vulnerabilities needed for missions

## Tutorial-Mission Alignment

### ✅ Good Alignment
- Tutorials teach tools that are available
- Mission hints provide contextual learning
- Tutorials cover all basic tools before missions require them

### 🟡 Minor Issues
- Some tutorials reference tools that might be mission-locked
- Mission hints could reference tutorial IDs for deeper learning

## Summary

**Overall Coherence**: 🟡 **Good, but needs fixes**

**Strengths**:
- Clear separation between free, premium, and exclusive tools (in design)
- Tutorials align with available tools
- Mission system is well-structured
- Shop system works correctly

**Critical Issues**:
- Mission exclusivity is broken (tools in repo that should be mission-only)
- Shop inventory is limited
- Mission hints reference unavailable tools

**Recommended Actions**:
1. **URGENT**: Remove mission-exclusive tools from repo
2. **HIGH**: Expand shop inventory with premium tools
3. **MEDIUM**: Fix mission hints to not reference unavailable tools
4. **LOW**: Add mission-specific servers for better story immersion

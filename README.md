### Initialization

- Initialization (good?)

### Features

- set name without auth
- not authenticated, save as guest object, save on login?
- name command in chat too
- unique names generator for alliances?
- autocomplete for /join with available user.alliance[] or the way socket keeps track of the rooms.
- autocomplete on file handling (edit etc)
- send more data from tool on save - should content be description?
- scan command (flags to show different info)
- user roles list /etc/groups/ /etc/password
- Virtual Sub-Root System - reveal root
- exploit kit upgrades (add different functionality to the kit)
- term.write system (also change everything to (eol?) and writeln)
- Better response system with colors - line breaks etc
- hidden files and a tool to find them (.files?) ls -a?
- another way to upgrade tools?
- shoult i be using socket.emit => callback instead of socket.on? With callback the user cant do anything while it works, which is good?
- tool information on server might not be the smartest.
- need to check vulnerable boolean on server.services on connection to server

### Improvements

- Guest restrictions (no alliances done - tool restrictions - what more?)
- Check over gameHandlers, they might not do what I want
- refactor data/.json handling/structure
- move fileData handling from login.js to fileSystem.js?
- improve cd to autocomplete full paths.
- test edit mode
- PROMPT! especially on SSH should be ip>
- .trim() command arguments?
- I should use utils more places getUsers etc
- I should set up auth checks with checkAuth instead. (Should this return a username? or is it enough with the req.socket?)

### Fixes

- some lines do not follow with a prompt.. can it be because of async? callbacks
  - set name
  - password_cracker downloaded successfully
  - logged out successfully

### Gameplay

- scan
- scan ip
  - security scanner
  - network sniffer
  - analyze data
- exploit
- password
- post:
  - rootkit, backdoors, etc.
  - steal data
  - network sniffer etc.
  - clean logs
- Scan for IPs
- Break into servers
- Extract resources or tools
- Install backdoors
- Clear logs
- Engage in cyber warfare with other players (optional PvP elements)
- hidden files and a tool to find them (.files?) ls -a? /root should be hidden so it shows (/)
- another way to upgrade tools?

### Structure

- add user tools to /bin or something
- refactor data/.json handling/structure
- move fileData handling from login.js to fileSystem.js?

### UI

- PROGRESS BARS!
- process dashboard!
- more commands
- list all kinds of info
- Better response system with colors - line breaks etc
- term.write system (also change everything to (eol?) and writeln)
- PROMPT! especially on SSH should be ip>

### GUI/TUI

- alternative login
- notifications
- dialog/modal tutorial
- ctrl + key to change windows chat mode / game log / etc++

### DB

- firebase? realtime. easy auth.
- Idle time should happens serverless?
- actual logs on the servers in internet.json
- MAKE EVERYTHING MORE REAL!

### Story

- start with scan? find portscanner free online.
- log into local network for direct practice
- log into neighbors network with sniffer (do fun stuff)
- Implement a mission system with different objectives (e.g., steal specific data, plant a backdoor).
- Add a storyline to guide players through the game.

### Names

- PacketStorm, ExploitMadness, ProtocolBuster, PacketScripter, sysAdmin, RootExploiter, KernelPanic, BackdoorScripter

### Other

- ai chat bots
- ai players
- command that creates new dynamic servers?
- other players have dynamic ports open?
- set up fake "wifi" / server for phishing etc
- make randomly generated file system for servers. ai?
- everything random! use deauth and some servers might never appear again.

### Economy

- Currency: Earned through hacking and mining
- Marketplace: Buy/sell tools, exploits, resources
- Crypto mining: Passive income generation

### Game Mechanics

- Exploration:
  - Users can use scanners to find new IPs to hack.
  - Some IPs are hidden and require advanced scanners or clues from other hacked servers.
- Hacking:
  - Different servers require different tools and techniques.
  - Players can level up and unlock more advanced tools.
  - Hacking can be done through mini-games or command sequences.
- Resource Management:
  - Players need to manage their resources effectively to progress.
  - Balance between active hacking and passive income generation (crypto mining).
- Security and Defense:
  - Servers can have different security measures.
  - Players can defend their own systems against AI or real player attacks (if PvP is enabled).
- Story and Missions:
  - Implement a mission system with different objectives (e.g., steal specific data, plant a backdoor).
  - Add a storyline to guide players through the game.

### Crypto

- Server Defenses:
  - Implement defense mechanisms on servers that can slow down or stop mining.
  - Players need to periodically re-hack or disable defenses.
- Multiple Miners:
  - Allow players to deploy multiple miners but with diminishing returns based on server resources.
- Resource Management:
  - Introduce resource balancing where over-mining a server can deplete it permanently or trigger defensive actions.
- Mining Efficiency Upgrades:
  - Players can invest in upgrades to make their mining operations more efficient.
- Competitive Mining:
  - Players can compete for control of high-value servers, introducing PvP elements where they can disrupt each other's mining operations.
- Dynamic Market:
  - Create a dynamic marketplace where mined cryptocurrency can fluctuate in value based on player activity.

### Resources and Their Uses

- CPU:
  - Usage: Most hacking tools and operations require CPU power. This represents the processing capability needed to perform tasks.
  - Drains:
    - Running hacking tools (e.g., password crackers, exploit kits).
    - Mining operations (e.g., crypto mining, bandwidth mining).
    - Intensive operations (e.g., DDoS attacks, data exfiltration).
- RAM:
  - Usage: Each tool or operation consumes a certain amount of RAM. The total RAM usage limits the number of simultaneous operations.
  - Drains:
    - Each active tool or operation will consume a portion of available RAM.
    - Running multiple tools concurrently can deplete RAM quickly, forcing players to manage their resources effectively.
- Bandwidth:
  - Usage: Bandwidth is used for data transfer operations. High-bandwidth tools or operations will require more of this resource.
  - Drains:
    - Data exfiltration (e.g., stealing data from a server).
    - High-traffic operations (e.g., DDoS attacks, running web shells).
    - Communication-heavy tools (e.g., network sniffers, signal interceptors).
- Data
  - Usage: Represents sensitive information that can be stolen and sold.
  - Stealing Data:
    - Use tools like data exfiltrators to steal data from servers.
    - Each server has a finite amount of data, and data theft reduces this resource.
  - Selling Data:
    - Players can sell stolen data on a marketplace for currency or crypto.
- Bandwidth
  - Usage: Can be stolen and utilized to increase the user's network capabilities.
  - Stealing Bandwidth:
    - Use tools like bandwidth miners to steal bandwidth from servers.
  - Applying Bandwidth:
    - Allows the user to run more bandwidth-intensive operations.
    - Can be used to enhance DDoS attacks or improve the efficiency of other tools.

### Mining Mechanics

For mining operations, it makes sense to use CPU resources primarily because mining is a CPU-intensive process. Here's a refined approach:

- Crypto Mining:
  - Resource Consumption: Uses CPU to mine cryptocurrency over time.
  - Resource Generation: Generates cryptocurrency passively.
  - Constraints: Limited by the CPU available on the compromised server and the user's available CPU and RAM.

### Additional Gameplay Mechanics

- Tool Activation and Resource Management:
  - Players must manage CPU, RAM, and bandwidth to optimize their hacking operations.
  - Running too many tools simultaneously can deplete resources, forcing strategic decisions.
- Resource Upgrades:
  - Allow players to upgrade their systems to increase CPU, RAM, and bandwidth capacity.
  - Upgrades can be purchased using in-game currency or cryptocurrency.
- Server Resource Management:
  - Each server has a limited amount of resources (CPU, bandwidth, data).
  - Overusing a server's resources can make it more difficult to hack or mine from.
- Defense Mechanisms:
  - Servers can have defense

mechanisms that activate when resources are overused.

- Players may need to hack or disable these defenses to continue their operations.
- Player vs. Player (PvP):
  - Introduce PvP elements where players can compete for control of high-value servers.
  - Players can disrupt each other's mining operations or defend their own servers.
- Balancing Resource Drains
  - CPU and RAM:
    - Each tool and operation should have a clear CPU and RAM cost.
    - More powerful tools should consume more resources, limiting the number of simultaneous operations.
  - Bandwidth:
    - Bandwidth-intensive operations should consume more bandwidth, requiring players to balance their network usage.
    - Stealing bandwidth can temporarily boost a player's capabilities but should be balanced against other operations.

### Attack

- Network Scanning
  - Tool: port_scanner
  - Action: Scan the network to identify active IP addresses and open ports.
  - Objective: Identify potential targets on the network.
- Service and Port Scanning
  - Tool: port_scanner
  - Action: Scan specific IP addresses for open ports and running services.
  - Objective: Identify services running on the target IP that are susceptible to weak passwords.
- Identify Vulnerabilities
  - Tool: security_scanner (optional)
  - Action: Scan the identified services for known vulnerabilities.
  - Objective: Confirm the presence of vulnerabilities such as weak passwords.
- Exploit Vulnerabilities
  - Tool: exploit_kit
  - Action: Use an exploit kit to gain initial access if vulnerabilities other than weak passwords are found.
  - Objective: Gain initial access to the target system.
- Brute Force Passwords
  - Tool: password_cracker
  - Action: Use the password cracker tool to brute force login credentials for the identified services.
  - Objective: Gain authenticated access to the target service.
- Elevate Privileges and Establish Persistence
  - Tool: rootkit
  - Action: Install a rootkit or other malware to maintain access.
  - Objective: Ensure continued access to the system.
- Post-Exploitation Actions
  - Tools: Various (e.g., data_exfiltrator, crypto_miner, network_sniffer)
  - Actions:
    - Data Exfiltration: Steal sensitive data.
    - Resource Stealing: Utilize the system's resources for mining cryptocurrency.
    - Browse Filesystem: Explore and manipulate files and directories.
    - Clean Logs: Use log_cleaner to erase traces of the attack.
  - Objective: Achieve the attacker's goals (data theft, resource exploitation, etc.).

This organized structure should help you manage the various aspects of your project more effectively.

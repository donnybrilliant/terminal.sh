- Initialization
- No user can be names user
- README is not present when initializing when authenticated
- Better response system with colors
- term.write system (also change everything to (eol?) and writeln)
- set name without auth
- not authenticated, save as guest object, save on login?

Now i want to think bigger about this! I want this to be a game with socketio. It should also have a chat. The chat should be initiated with "chat" command, bypass all other commands and have own commands starting with / or : .
There should be one general room, but the user can enter a command /alliance (or :alliance) with one or more existing usernames to create a private room for those users.
Everything should also be written to a json file. To log.

For the game side, im still a bit unsure what it should be. But it is definitely inspired by uplink and hacknet. So i want to create a fake json internet, the user can scan for ips, they have to break in, find new tools they save to their user object. When they are logged in, maybe they can steal resources from that server they can use for other attacks etc?
Im open for as many crazy ideas as possible. The sky is the limit.

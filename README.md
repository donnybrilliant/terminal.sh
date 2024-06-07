- Initialization
- Guest resctrictions (no alliances etc)
- fix alliance (should not work without args)
- README is not present when initializing when authenticated
- Better response system with colors - line breaks etc
- term.write system (also change everything to (eol?) and writeln)
- set name without auth
- not authenticated, save as guest object, save on login?
- name command in chat too
- unique names generator for alliances?
- autocomplete for /join with available user.alliance[] or the way socket keeps track of the rooms.
- tab suggestions on multiple lines causes rerender on new lines...

Now i want to think bigger about this! I want this to be a game with socketio. It should also have a chat. The chat should be initiated with "chat" command, bypass all other commands and have own commands starting with / or : .
There should be one general room, but the user can enter a command /alliance (or :alliance) with one or more existing usernames to create a private room for those users.
Everything should also be written to a json file. To log.

For the game side, im still a bit unsure what it should be. But it is definitely inspired by uplink and hacknet. So i want to create a fake json internet, the user can scan for ips, they have to break in, find new tools they save to their user object. When they are logged in, maybe they can steal resources from that server they can use for other attacks etc?
Im open for as many crazy ideas as possible. The sky is the limit.

Its not giving me double messages anymore!!

So now a couple of things:

- on exit it emits joinGeneral - we should have an exit here? Same as with disconnect actually.
- I want to change the way i log things. there should be a json file for each room in "messages", which contains all the messages for each namespace. one general, one for each alliance as they are created. They should not contain any join or exit messages. Only the messages sent from the user.
- I want to have a list of all the users that are under filesystem.json...users or all the users in users.json which shows when i type the command :list or :users (or starting with /). Then i also need to show who of these users are online. Guest will be a bit special, since many connections might be named that, but you should list only the current connections of these, not save the older ones that are disconnected.

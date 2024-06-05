import { term, loginManager, socket } from "./index.js";

let chatMode = false;
let currentChatRoom = "general";

export function setupChat() {
  const user = loginManager.getUsername() || "Guest";

  socket.on("message", (message) => {
    term.write(`\r\n${message}`);
  });

  // Initialize chat by joining the general room
  socket.emit("joinGeneral", user);
}

export function handleChatCommand(command) {
  const user = loginManager.getUsername() || "Guest";

  if (command.startsWith("/") || command.startsWith(":")) {
    const parts = command.split(" ");
    const cmd = parts[0].substring(1);
    const args = parts.slice(1);

    if (cmd === "alliance") {
      socket.emit("createAlliance", {
        usernames: args,
        creator: user,
      });
    } else if (cmd === "exit") {
      chatMode = false;
      currentChatRoom = "general";
      socket.emit("joinGeneral", user); // Rejoin general room - this might not be wanted..
      return "Exiting chat mode.";
    } else {
      socket.emit("chatMessage", {
        room: currentChatRoom,
        message: command,
        username: user,
      });
    }
  } else {
    socket.emit("chatMessage", {
      room: currentChatRoom,
      message: command,
      username: user,
    });
  }
}

export function initializeChat() {
  chatMode = true;
  const user = loginManager.getUsername() || "Guest";
  socket.emit("joinGeneral", user);
}

export function isInChatMode() {
  return chatMode;
}

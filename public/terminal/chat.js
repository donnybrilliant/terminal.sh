//import { io } from "/socket.io/socket.io.js";
import { term, loginManager } from "./index.js";

const socket = io();
let chatMode = false;
let currentChatRoom = "general";

const user = loginManager.getUsername() || "Guest";

socket.on("message", (message) => {
  term.writeln(message);
});

// Initialize chat by joining the general room
socket.emit("joinGeneral", user); // Replace with actual username

export function handleChatCommand(command) {
  if (command.startsWith("/") || command.startsWith(":")) {
    const parts = command.split(" ");
    const cmd = parts[0].substring(1);
    const args = parts.slice(1);

    if (cmd === "alliance") {
      socket.emit("createAlliance", {
        usernames: args,
        creator: user, // Replace with actual username
      });
    } else if (cmd === "exit") {
      chatMode = false;
      currentChatRoom = "general";
      socket.emit("joinGeneral", user); // Rejoin general room
      return "Exiting chat mode.";
    } else {
      socket.emit("chatMessage", {
        room: currentChatRoom,
        message: command,
        username: user, // Replace with actual username
      });
    }
  } else {
    socket.emit("chatMessage", {
      room: currentChatRoom,
      message: command,
      username: user, // Replace with actual username
    });
  }
}

export function initializeChat() {
  chatMode = true;
  socket.emit("joinGeneral", user); // Replace with actual username
}

export function isInChatMode() {
  return chatMode;
}

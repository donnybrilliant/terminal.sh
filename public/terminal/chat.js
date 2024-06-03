//import { io } from "/socket.io/socket.io.js";
import { term } from "./index.js"; // Adjust the import path as needed

let isChatMode = false;

export function isInChatMode() {
  return isChatMode;
}

export function enterChatMode() {
  isChatMode = true;
}

export function exitChatMode() {
  isChatMode = false;
}

export function initializeChat() {
  const socket = io("/admin");
  const room = "default-room"; // Default single room

  socket.on("connect", () => {
    socket.emit("join", { room: room });
  });

  enterChatMode();

  socket.on("load messages", (messages) => {
    messages.forEach((message) => {
      term.writeln(`${message.username}: ${message.msg}`);
    });
    term.scrollToBottom();
  });

  socket.on("chat message", (msg) => {
    term.writeln(msg);
    term.scrollToBottom();
  });

  term.onKey((e) => {
    const char = e.key;
    if (char === "\r") {
      // Enter key
      const input = term.buffer.active
        .getLine(term.buffer.active.cursorY)
        .translateToString(true)
        .trim();
      console.log(input);
      if (input === ":exit") {
        exitChatMode();
        term.write("\r\nExited chat mode.\r\n$ ");
      } else if (input.length > 0) {
        socket.emit("chat message", {
          room: room,
          username: "User",
          msg: input,
        });
        term.write("\r\n");
      }
    }
  });
}

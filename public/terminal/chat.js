import { term, loginManager } from "./index.js";

// Chat-specific socket
let chatNamespace;

let chatMode = false;
let currentChatRoom = "general";

export function setupChat() {
  chatNamespace = io("/chat");

  const user = loginManager.getUsername() || "Guest";

  chatNamespace.on("message", (message) => {
    // Clear the current line and move cursor to the beginning
    term.write(`\r\x1b[2K\r`);
    term.write(`${message}\r\n`);
    renderPrompt();
  });

  chatNamespace.emit("joinGeneral", user);
}

export function handleChatCommand(command) {
  const user = loginManager.getUsername() || "Guest";

  if (command.startsWith("/") || command.startsWith(":")) {
    const parts = command.split(" ");
    const cmd = parts[0].substring(1);
    const args = parts.slice(1);

    if (cmd === "alliance") {
      chatNamespace.emit("createAlliance", {
        usernames: args,
        creator: user,
      });
    } else if (cmd === "exit") {
      chatMode = false;
      currentChatRoom = "general";
      chatNamespace.emit("joinGeneral", user);
      renderPrompt();
      return "Exiting chat mode.";
    } else {
      chatNamespace.emit("chatMessage", {
        room: currentChatRoom,
        message: command,
        username: user,
      });
    }
  } else {
    chatNamespace.emit("chatMessage", {
      room: currentChatRoom,
      message: command,
      username: user,
    });
  }
  renderPrompt();
}

export function initializeChat() {
  if (!chatNamespace) {
    setupChat();
  }

  chatMode = true;
  const user = loginManager.getUsername() || "Guest";
  chatNamespace.emit("joinGeneral", user);
  renderPrompt();
}

export function isInChatMode() {
  return chatMode;
}

function renderPrompt() {
  const user = loginManager.getUsername();
  const prompt = user ? `${user}> ` : "> ";
  term.write(prompt);
}

import { term, loginManager } from "./index.js";

// Chat-specific socket
let chatNamespace;
let chatMode = false;
let currentChatRoom = "general";

// Chat command map
const chatCommandMap = {
  alliance: (args, user) => {
    chatNamespace.emit("createAlliance", { usernames: args, creator: user });
  },
  join: (args) => {
    const room = args[0];
    if (room) {
      currentChatRoom = room;
      chatNamespace.emit("joinRoom", room);
    }
  },
  alliances: () => {
    chatNamespace.emit("listAlliances");
  },
  exit: (user) => {
    chatMode = false;
    currentChatRoom = "general";
    chatNamespace.emit("exit", user);
    renderPrompt();
    return "Exiting chat mode.";
  },
  list: () => {
    chatNamespace.emit("listUsers");
  },
  users: () => {
    chatNamespace.emit("listUsers");
  },
};

export function getChatCommandList() {
  return Object.keys(chatCommandMap).map((cmd) => `/${cmd}`);
}

export function setupChat() {
  if (!chatNamespace) {
    chatNamespace = io("/chat");

    chatNamespace.on("message", (message) => {
      // Clear the current line and move cursor to the beginning
      term.write(`\r\x1b[2K\r`);
      term.write(`${message}\r\n`);
      renderPrompt();
    });

    chatNamespace.on("userList", (users) => {
      // Clear the current line and move cursor to the beginning
      term.write(`\r\x1b[2K\r`);
      term.write(`\r\nUsers:\r\n${users.join("\r\n")}\r\n`);
      renderPrompt();
    });
  }

  const user = loginManager.getUsername() || "Guest";
  chatNamespace.emit("joinGeneral", user);
}

export function handleChatCommand(command) {
  const user = loginManager.getUsername() || "Guest";

  if (command.startsWith("/") || command.startsWith(":")) {
    const parts = command.split(" ");
    const cmd = parts[0].substring(1);
    const args = parts.slice(1);

    const commandFunction = chatCommandMap[cmd];
    if (commandFunction) {
      return commandFunction(args, user);
    } else {
      return `Unknown chat command: ${cmd}`;
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

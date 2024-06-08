import { term, loginManager } from "../terminal/index.js";
import {
  saveCommandBuffer,
  restoreCommandBuffer,
} from "../terminal/keyInputHandler.js";

// Chat-specific socket
let chatNamespace;
let chatMode = false;
let currentChatRoom = "general";

// Chat command map
const chatCommandMap = {
  alliance: (args, user) => {
    if (args.length === 0) {
      term.write(`\r\nNo username(s) provided.\r\n`);
    } else {
      chatNamespace.emit("createAlliance", { usernames: args, creator: user });
    }
  },
  join: (args) => {
    const room = args[0];
    if (room) {
      chatNamespace.emit("joinRoom", room);
    } else {
      chatNamespace.emit("listAlliances");
    }
  },
  leave: () => {
    if (currentChatRoom === "general") {
      term.write(
        `\r\nYou are in the general room. Use ':exit' to leave the chat.\r\n`
      );
      renderPrompt();
    } else {
      chatNamespace.emit("leaveRoom");
    }
  },
  alliances: () => {
    chatNamespace.emit("listAlliances");
  },
  exit: (user) => {
    chatMode = false;
    chatNamespace.emit("exit", user);
    chatNamespace = null; // Ensure the namespace is set to null for reconnection
    term.write(`\r\nExited chat.\r\n`);
    renderPrompt();
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
      const state = saveCommandBuffer();
      // Clear the current line and move cursor to the beginning
      term.write(`\r\x1b[2K\r`);
      term.write(`${message}\r\n`);
      restoreCommandBuffer(term, state);
    });

    chatNamespace.on("userList", (users) => {
      // Clear the current line and move cursor to the beginning
      term.write(`\r\x1b[2K\r`);
      term.write(`\r\nUsers:\r\n${users.join("\r\n")}\r\n`);
      renderPrompt();
    });

    chatNamespace.on("listAlliances", (alliances) => {
      const state = saveCommandBuffer();
      if (alliances.length > 0) {
        term.write(`\r\x1b[2K\r`);
        term.write(`\r\nAlliances:\r\n${alliances.join("\r\n")}\r\n`);
      } else {
        term.write(`\r\x1b[2K\r`);
        term.write(`\r\nYou have no alliances.\r\n`);
      }
      //renderPrompt() - save/restoreCommandBuffer vs renderPrompt here?
      restoreCommandBuffer(term, state);
    });

    // Listen for room change confirmation and update currentChatRoom
    chatNamespace.on("roomChanged", (newRoom) => {
      currentChatRoom = newRoom;
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
  term.write(`\r\nWelcome to the chat! Type ':exit' to leave chat mode.\r\n`);
}

export function isInChatMode() {
  return chatMode;
}

function renderPrompt() {
  const user = loginManager.getUsername();
  const prompt = isInChatMode() ? `${user}> ` : `${user}$ `;
  term.write(prompt);
}

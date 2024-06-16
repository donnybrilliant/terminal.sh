// commandProcessor.js
import { commands } from "./shell.js";
import { term, socket, loginManager } from "./index.js";
import { loadtest, chars, hack, startMatrix, getClientInfo } from "./random.js";
import {
  editFile,
  saveEdits,
  exitEdit,
  isInEditMode,
  appendToEditedContent,
  getEditedContent,
} from "./edit.js";
import {
  initializeChat,
  isInChatMode,
  handleChatCommand,
} from "../chat/index.js";
import { fileData } from "./fileSystem.js";
import {
  currentSSHSession,
  isInSSHMode,
  handleSSHCommand,
} from "../ssh/index.js";

// Base Command map
const baseCommandMap = {
  nano: (args) => editFile(args[0]),
  vi: (args) => editFile(args[0]),
  edit: (args) => editFile(args[0]),
  ls: (args) => commands.ls(args),
  cd: (args) => {
    if (args.length !== 1) {
      return "Usage: cd <folder> or ..";
    }
    return commands.cd(args[0]);
  },
  cat: (args) => {
    if (args.length !== 1) {
      return "Usage: cat <filename>";
    }
    return commands.cat(args[0]);
  },
  pwd: () => commands.pwd(),
  help: () => commands.help(),
  loadtest: () => loadtest(term),
  chars: () => chars(term),
  hack: () => hack(term),
  matrix: () => startMatrix(term),
  info: () => getClientInfo(),
  name: async (args) => {
    if (args.length !== 1) {
      return "Usage: name <newName>";
    }
    await loginManager.setName(args[0]);
    return "";
  },
  rm: (args) => commands.rm(args),
  clear: () => commands.clear(),
  mkdir: (args) => {
    if (args.length !== 1) {
      return "Usage: mkdir <dirname>";
    }
    return commands.mkdir(args[0]);
  },
  touch: (args) => {
    if (args.length !== 1) {
      return "Usage: touch <filename>";
    }
    return commands.touch(args[0]);
  },
  cp: (args) => {
    if (args.length !== 2) {
      return "Usage: cp <source> <destination>";
    }
    return commands.cp(args[0], args[1]);
  },
  mv: (args) => {
    if (args.length !== 2) {
      return "Usage: mv <source> <destination>";
    }
    return commands.mv(args[0], args[1]);
  },
  hola: () => "hello",
  chat: () => {
    initializeChat();
    return ""; // Remove the "Welcome to the chat" message from here
  },
  login: async (args) => {
    if (args.length < 2) {
      return "Usage: login <username> <password>";
    }
    return await loginManager.login(args[0], args[1]);
  },
  logout: async () => await loginManager.logout(),
  scan: (args) => {
    if (args.length === 0) {
      const username = loginManager.getUsername() || "Guest";
      socket.emit("scanInternet", { username });
      return "Scanning internet for IP addresses...";
    } else if (args.length === 1) {
      const username = loginManager.getUsername() || "Guest";
      socket.emit("scanIP", { username, targetIP: args[0] });
      return `Scanning IP ${args[0]} for services...`;
    } else {
      return "Usage: scan <targetIP>";
    }
  },
  hackIP: (args) => {
    if (args.length !== 1) {
      return "Usage: hackIP <targetIP>";
    }
    const username = loginManager.getUsername() || "Guest";
    socket.emit("hackIP", { username, targetIP: args[0] });
    return `Attempting to hack IP ${args[0]}...`;
  },
  mine: (args) => {
    if (args.length !== 1) {
      return "Usage: mine <targetIP>";
    }
    const username = loginManager.getUsername() || "Guest";
    const targetIP = args[0];
    socket.emit("startMining", { username, targetIP });
    return `Mining IP ${targetIP}...`;
  },
  download: (args) => {
    const username = loginManager.getUsername() || "Guest";

    if (args.length === 2) {
      // Original format: download ip toolName
      const [targetIP, toolName] = args;
      socket.emit("download", {
        username,
        targetIP,
        toolName,
      });
      return `Downloading ${toolName} from IP ${targetIP}...`;
    } else if (args.length === 1) {
      // New format: download filePath
      const filePath = args[0];
      socket.emit("download", {
        username,
        filePath,
      });
      return `Downloading file from ${filePath}...`;
    } else {
      return "Usage: download <targetIP> <toolName> or download <filePath>";
    }
  },
  server: () => {
    socket.emit("requestHardwareInfo");
    socket.on("hardwareInfo", (data) => {
      console.log("Received hardware info:", data);
      // Use this information as needed
    });
    return "Hardware info received. Check the console.";
  },
  ssh: (args) => {
    if (args.length !== 1) {
      return "Usage: ssh <targetIP>";
    }
    const targetIP = args[0];
    const username = loginManager.getUsername();
    socket.emit("ssh", { username, targetIP });
    return `Connecting to ${targetIP}...`;
  },
};

// Tool-specific Command map
const toolCommandMap = {
  password_sniffer: (args) => {
    // fix better args handling when empty string
    if (args.length !== 1 || args[0] === "") {
      return "Usage: password_sniffer <targetIP>";
    }
    const username = loginManager.getUsername() || "Guest";
    socket.emit("password_sniffer", { username, targetIP: args[0] });
    return `Attempting to sniff password on IP ${args[0]}...`;
  },
  ssh_exploit: (args) => {
    if (args.length !== 1 || args[0] === "") {
      return "Usage: ssh_exploit <targetIP>";
    }
    const username = loginManager.getUsername() || "Guest";
    socket.emit("ssh_exploit", { username, targetIP: args[0] });
    return `Attempting to exploit SSH on IP ${args[0]}...`;
  },
  user_enum: (args) => {
    if (args.length !== 0) {
      return "Usage: user_enum";
    }
    const username = loginManager.getUsername() || "Guest";
    const targetIP = currentSSHSession.targetIP;
    if (!targetIP) {
      return "No active SSH session.";
    }
    socket.emit("user_enum", { username, targetIP });
    return `Enumerating users on IP ${targetIP}...`;
  },
  password_cracker: (args) => {
    if (args.length !== 1 || args[0] === "") {
      return "Usage: password_cracker <role>";
    }
    const username = loginManager.getUsername() || "Guest";
    const targetIP = currentSSHSession.targetIP;
    if (!targetIP) {
      return "No active SSH session.";
    }
    socket.emit("password_cracker", {
      username,
      targetIP,
      role: args[0],
    });
    return `Attempting to crack password for role ${args[0]} on IP ${targetIP}...`;
  },
  rootkit: (args) => {
    if (args.length !== 1 || args[0] === "") {
      return "Usage: rootkit <role>";
    }
    const username = loginManager.getUsername() || "Guest";
    const targetIP = currentSSHSession.targetIP;
    if (!targetIP) {
      return "No active SSH session.";
    }
    socket.emit("rootkit", {
      username,
      targetIP,
      role: args[0],
    });
    return `Initializing rootkit for role ${args[0]} on IP ${targetIP}...`;
  },
};

export function getCombinedCommandMap() {
  const username = loginManager.getUsername();
  let userTools = [];

  if (
    username &&
    fileData.home.users[username] &&
    fileData.home.users[username].bin
  ) {
    userTools = Object.keys(fileData.home.users[username].bin);
  }

  const combinedCommandMap = { ...baseCommandMap };
  userTools.forEach((tool) => {
    if (toolCommandMap[tool]) {
      combinedCommandMap[tool] = toolCommandMap[tool];
    }
  });

  return combinedCommandMap;
}

export function getCommandList() {
  const combinedCommandMap = getCombinedCommandMap();
  return Object.keys(combinedCommandMap);
}

export default async function processCommand(command) {
  const [cmd, ...args] = command.split(" ");
  const combinedCommandMap = getCombinedCommandMap();

  // Handle chat mode
  if (isInChatMode()) {
    return handleChatCommand(command);
  }

  // Handle SSH mode
  if (isInSSHMode()) {
    return await handleSSHCommand(command);
  }

  // Check if the system is in edit mode
  if (isInEditMode()) {
    if (cmd.trim() === ":save") {
      return saveEdits(getEditedContent().trim());
    } else if (cmd.trim() === ":exit") {
      return exitEdit();
    } else {
      appendToEditedContent(cmd); // Add the user input to editedContent
      return "";
    }
  }

  // If not in edit mode, proceed with normal command processing
  const commandFunction = combinedCommandMap[cmd];
  if (commandFunction) {
    return await commandFunction(args);
  } else {
    return `Unknown command: ${cmd}`;
  }
}

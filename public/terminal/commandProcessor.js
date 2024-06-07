import { commands } from "./shell.js";
import { term } from "./index.js";
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
import { loginManager } from "./index.js";

// Command map
const commandMap = {
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
  name: (args) => commands.name(args.join(" ")),
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
};

export function getCommandList() {
  return Object.keys(commandMap);
}

/**
 * Main function to process terminal commands.
 * Delegates work to individual command functions from the command map.
 *
 * @param {string} command - The command entered by the user.
 * @returns {string} - The output of the command.
 */
export default async function processCommand(command) {
  const [cmd, ...args] = command.split(" ");

  // Handle chat mode
  if (isInChatMode()) {
    return handleChatCommand(command);
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
  const commandFunction = commandMap[cmd];
  if (commandFunction) {
    return await commandFunction(args);
  } else {
    return `Unknown command: ${cmd}`;
  }
}

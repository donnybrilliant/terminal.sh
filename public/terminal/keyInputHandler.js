// keyInputHandler.js

import { stopMatrix } from "./random.js";
import { loginManager } from "./index.js";
import { isInEditMode } from "./edit.js";
import {
  isInChatMode,
  handleChatCommand,
  getChatCommandList,
} from "../chat/index.js";
import {
  isInSSHMode,
  handleSSHCommand,
  renderSSHPrompt,
} from "../ssh/index.js";
import { getCommandList } from "./commandProcessor.js";
import {
  getCurrentPath,
  getDirectoryNames,
  getFileNames,
} from "./fileSystem.js";

let commandBuffer = "";
let cursorPosition = 0;

let suggestions = [];
let isShowingSuggestions = false;

const mainCommandHistory = [];
const chatCommandHistory = [];
const sshCommandHistory = [];
let mainHistoryIndex = -1;
let chatHistoryIndex = -1;
let sshHistoryIndex = -1;

let availableCommands;
let chatCommands;

function initializeCommandLists() {
  availableCommands = getCommandList();
  chatCommands = getChatCommandList();
}

export function updateCommandList() {
  availableCommands = getCommandList();
}

function clearSuggestions(term) {
  if (isShowingSuggestions) {
    term.write(`\x1b[1B\r\x1b[2K`); // Move down one line and clear it
    term.write(`\x1b[1A\r`); // Move back up
    isShowingSuggestions = false;
  }
}

function displaySuggestions(term, suggestions) {
  if (suggestions.length > 0) {
    term.write(`\x1b[1B\r\x1b[2K`); // Move down one line and clear it
    term.write(`${suggestions.join("  ")}\r`); // Display suggestions
    term.write(`\x1b[1A\r`); // Move back up
    isShowingSuggestions = true;
  }
}

function render(term) {
  const user = loginManager.getUsername();
  const prompt = isInChatMode()
    ? `${user}> `
    : isInSSHMode()
    ? `${user}@${getCurrentPath(true)}$ `
    : `${user}$ `;
  term.write(`\r\x1b[2K\r${prompt}${commandBuffer}`);
  term.write(`\x1b[${cursorPosition + prompt.length + 1}G`);
  if (isShowingSuggestions) {
    term.write(`\x1b[1B\r${suggestions.join("  ")}\r`);
    term.write(`\x1b[1A\r`);
  }
}

export function saveCommandBuffer() {
  return { commandBuffer, cursorPosition };
}

export function restoreCommandBuffer(term, state) {
  commandBuffer = state.commandBuffer;
  cursorPosition = state.cursorPosition;
  render(term);
}

export default async function handleKeyInput(
  { key, domEvent },
  term,
  processCommand
) {
  if (!availableCommands) {
    initializeCommandLists();
  }

  const keyCode = domEvent.keyCode || domEvent.which;
  const history = isInChatMode()
    ? chatCommandHistory
    : isInSSHMode()
    ? sshCommandHistory
    : mainCommandHistory;
  let historyIndex = isInChatMode()
    ? chatHistoryIndex
    : isInSSHMode()
    ? sshHistoryIndex
    : mainHistoryIndex;
  const user = loginManager.getUsername();
  const prompt = isInChatMode()
    ? `${user}> `
    : isInSSHMode()
    ? `${user}@${getCurrentPath(true)}$ `
    : `${user}$ `;

  if (keyCode === 37) {
    // Left arrow
    if (cursorPosition > 0) {
      cursorPosition--;
      term.write("\x1b[D");
    }
    return;
  }

  if (keyCode === 39) {
    // Right arrow
    if (cursorPosition < commandBuffer.length) {
      cursorPosition++;
      term.write("\x1b[C");
    }
    return;
  }

  if (keyCode === 38) {
    // Up arrow
    if (historyIndex > 0) {
      historyIndex--;
      commandBuffer = history[historyIndex] || "";
      cursorPosition = commandBuffer.length;
      clearSuggestions(term);
      render(term);
    } else if (historyIndex === 0) {
      commandBuffer = history[historyIndex] || "";
      cursorPosition = commandBuffer.length;
      clearSuggestions(term);
      render(term);
    }
    if (isInChatMode()) {
      chatHistoryIndex = historyIndex;
    } else if (isInSSHMode()) {
      sshHistoryIndex = historyIndex;
    } else {
      mainHistoryIndex = historyIndex;
    }
    return;
  }

  if (keyCode === 40) {
    // Down arrow
    if (historyIndex < history.length - 1) {
      historyIndex++;
      commandBuffer = history[historyIndex] || "";
      cursorPosition = commandBuffer.length;
      clearSuggestions(term);
      render(term);
    } else if (historyIndex === history.length - 1) {
      historyIndex++;
      commandBuffer = "";
      cursorPosition = 0;
      clearSuggestions(term);
      render(term);
    }
    if (isInChatMode()) {
      chatHistoryIndex = historyIndex;
    } else if (isInSSHMode()) {
      sshHistoryIndex = historyIndex;
    } else {
      mainHistoryIndex = historyIndex;
    }
    return;
  }

  if (key === "Backspace" || keyCode === 8) {
    if (cursorPosition > 0) {
      commandBuffer =
        commandBuffer.slice(0, cursorPosition - 1) +
        commandBuffer.slice(cursorPosition);
      cursorPosition--;
      clearSuggestions(term);
      render(term);
    }
    return;
  }

  // Handle Tab key press for command auto-completion

  if (key === "Tab" || keyCode === 9) {
    const [command, ...args] = commandBuffer.split(" ");
    if (command === "cd") {
      const currentPath = args.join(" ");
      const directories = isInSSHMode()
        ? getDirectoryNames(true)
        : getDirectoryNames();
      const possibleDirs = directories.filter((dir) =>
        dir.startsWith(currentPath)
      );

      clearSuggestions(term);

      if (possibleDirs.length === 1) {
        commandBuffer = `cd ${possibleDirs[0]}`;
        cursorPosition = commandBuffer.length;
        render(term);
      } else if (possibleDirs.length > 1) {
        suggestions = possibleDirs;
        displaySuggestions(term, possibleDirs);
        term.write(`\x1b[${cursorPosition + prompt.length + 1}G`);
      } else {
        suggestions = [];
        render(term);
      }

      domEvent.preventDefault();
      return;
    } else if (command === "cat") {
      const currentPath = args.join(" ");
      const files = isInSSHMode() ? getFileNames(true) : getFileNames();
      const possibleFiles = files.filter((file) =>
        file.startsWith(currentPath)
      );

      clearSuggestions(term);

      if (possibleFiles.length === 1) {
        commandBuffer = `cat ${possibleFiles[0]}`;
        cursorPosition = commandBuffer.length;
        render(term);
      } else if (possibleFiles.length > 1) {
        suggestions = possibleFiles;
        displaySuggestions(term, possibleFiles);
        term.write(`\x1b[${cursorPosition + prompt.length + 1}G`);
      } else {
        suggestions = [];
        render(term);
      }

      domEvent.preventDefault();
      return;
    } else {
      const commands = isInChatMode() ? chatCommands : availableCommands;
      const possibleCommands = commands.filter((cmd) =>
        cmd.startsWith(commandBuffer)
      );

      clearSuggestions(term);

      if (possibleCommands.length === 1) {
        commandBuffer = possibleCommands[0] + " ";
        cursorPosition = commandBuffer.length;
        render(term);
      } else if (possibleCommands.length > 1) {
        suggestions = possibleCommands;
        displaySuggestions(term, possibleCommands);
        term.write(`\x1b[${cursorPosition + prompt.length + 1}G`);
      } else {
        suggestions = [];
        render(term);
      }
      domEvent.preventDefault();
      return;
    }
  }

  // Handle Enter key press
  if (keyCode === 13) {
    if (commandBuffer.trim() === "") {
      render(term); // Just render the prompt again if the command buffer is empty
      return;
    }

    if (isInChatMode()) {
      handleChatCommand(commandBuffer);
      chatCommandHistory.push(commandBuffer);
      chatHistoryIndex = chatCommandHistory.length;
      commandBuffer = "";
      cursorPosition = 0;
      clearSuggestions(term);
      render(term); // Render the chat prompt
      return;
    } else if (isInSSHMode()) {
      const output = await handleSSHCommand(commandBuffer);
      if (output) {
        term.write(`\r\n${output}`);
      }
      sshCommandHistory.push(commandBuffer);
      sshHistoryIndex = sshCommandHistory.length;
      commandBuffer = "";
      cursorPosition = 0;
      clearSuggestions(term);
      term.write("\r\n"); // Ensure we move to a new line
      render(term);
      term.scrollToBottom();
      return;
    } else {
      const output = await processCommand(commandBuffer);
      if (!isInEditMode() && output) {
        term.write(`\r\n${output}`);
      }
      mainCommandHistory.push(commandBuffer);
      mainHistoryIndex = mainCommandHistory.length;
      commandBuffer = "";
      cursorPosition = 0;
      clearSuggestions(term);
      term.write("\r\n"); // Ensure we move to a new line
      render(term);
      term.scrollToBottom();
      return;
    }
  }

  // Handle Ctrl + C key press
  if (domEvent.ctrlKey && domEvent.key === "c") {
    stopMatrix();
    term.write("\r\nInterrupted");
    commandBuffer = ""; // Reset the command buffer
    cursorPosition = 0;
    clearSuggestions(term);
    render(term); // Render the prompt
    return;
  }

  // For regular key presses, insert the character at the cursor position
  if (key.length === 1) {
    commandBuffer =
      commandBuffer.slice(0, cursorPosition) +
      key +
      commandBuffer.slice(cursorPosition);
    cursorPosition++;
    clearSuggestions(term);
    render(term);
  }
}

import { stopMatrix } from "./random.js";
import { loginManager } from "./index.js";
import { isInEditMode } from "./edit.js";
import { isInChatMode, handleChatCommand, getChatCommandList } from "./chat.js";
import { getCommandList } from "./commandProcessor.js"; // Import getCommandList

// Buffer to hold the current command being typed by the user.
let commandBuffer = "";
let cursorPosition = 0;

// Command history for the main terminal and chat
const mainCommandHistory = [];
const chatCommandHistory = [];
let mainHistoryIndex = -1;
let chatHistoryIndex = -1;

// List of available commands for auto-completion
const availableCommands = getCommandList();
const chatCommands = getChatCommandList();

/**
 * Renders the terminal prompt and command buffer.
 *
 * @param {Object} term - The xterm.js terminal object.
 */
function render(term) {
  const user = loginManager.getUsername();
  const prompt = isInChatMode() ? `${user}> ` : `${user}$ `;
  term.write(`\r\x1b[2K\r${prompt}${commandBuffer}`);
  term.write(`\x1b[${cursorPosition + prompt.length + 1}G`);
}

/**
 * Handles individual key inputs from the user for the terminal.
 *
 * @param {Object} param0 - Destructured parameter object.
 * @param {string} param0.key - The character representation of the key pressed.
 * @param {Object} param0.domEvent - The original key event object.
 * @param {Object} term - The xterm.js terminal object.
 * @param {function} processCommand - The function to process the full command once Enter is pressed.
 */
export default async function handleKeyInput(
  { key, domEvent },
  term,
  processCommand
) {
  const keyCode = domEvent.keyCode || domEvent.which;
  const history = isInChatMode() ? chatCommandHistory : mainCommandHistory;
  let historyIndex = isInChatMode() ? chatHistoryIndex : mainHistoryIndex;

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
      render(term);
    } else if (historyIndex === 0) {
      commandBuffer = history[historyIndex] || "";
      cursorPosition = commandBuffer.length;
      render(term);
    }
    if (isInChatMode()) {
      chatHistoryIndex = historyIndex;
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
      render(term);
    } else if (historyIndex === history.length - 1) {
      historyIndex++;
      commandBuffer = "";
      cursorPosition = 0;
      render(term);
    }
    if (isInChatMode()) {
      chatHistoryIndex = historyIndex;
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
      render(term);
    }
    return;
  }

  // Handle Tab key press for command auto-completion
  if (key === "Tab" || keyCode === 9) {
    const commands = isInChatMode() ? chatCommands : availableCommands;
    const possibleCommands = commands.filter((cmd) =>
      cmd.startsWith(commandBuffer)
    );
    if (possibleCommands.length === 1) {
      commandBuffer = possibleCommands[0];
      cursorPosition = commandBuffer.length;
      render(term);
    } else if (possibleCommands.length > 1) {
      term.write(`\r\n${possibleCommands.join("  ")}\r\n`);
      render(term);
    }
    return;
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
      render(term); // Render the chat prompt
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
    render(term);
  }
}

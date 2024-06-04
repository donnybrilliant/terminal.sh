import { stopMatrix } from "./random.js";
import { loginManager } from "./index.js";
import { isInEditMode } from "./edit.js";
import { isInChatMode } from "./chat.js";

// Buffer to hold the current command being typed by the user.
let commandBuffer = "";

/**
 * Gets the current value of the command buffer.
 *
 * @returns {string} - The current value of the command buffer.
 */
export function getCommandBuffer() {
  return commandBuffer;
}

/**
 * Sets a new value for the command buffer.
 *
 * @param {string} value - The value to set for the command buffer.
 */
export function setCommandBuffer(value) {
  commandBuffer = value;
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
  const keyCode = domEvent.keyCode;

  // Handle backspace key press
  if (keyCode === 8 && commandBuffer.length > 0) {
    commandBuffer = commandBuffer.slice(0, -1);
    term.write("\b \b"); // Erase the last character
  }

  // Handle Enter key press
  else if (keyCode === 13) {
    const output = await processCommand(commandBuffer);
    if (isInEditMode()) {
      term.write(`\r\n${output}`);
    } else {
      // If not in edit mode, add a new line and prompt
      const user = loginManager.getUsername();
      if (output) {
        // Only write if output is not empty
        term.write(`\r\n${output}\r\n${user}$ `);
      }
    }
    commandBuffer = "";
    term.scrollToBottom();
  }

  // Handle Ctrl + C key press
  else if (domEvent.ctrlKey && domEvent.key === "c") {
    stopMatrix();
    term.write("\r\nInterrupted\r\n$ ");
    commandBuffer = ""; // Reset the command buffer
  }

  // For regular key presses, append the character to the command buffer and write to terminal
  else {
    commandBuffer += key;
    term.write(key);
  }
}

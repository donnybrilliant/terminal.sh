import { stopMatrix } from "./random.js";
import { getName } from "./fileSystem.js";
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
export default function handleKeyInput(
  { key, domEvent },
  term,
  processCommand
) {
  const keyCode = domEvent.keyCode;

  // Handle backspace key press
  if (keyCode === 8) {
    if (commandBuffer.length > 0) {
      commandBuffer = commandBuffer.slice(0, -1);
      term.write("\b \b"); // Erase the last character
    }
    return; // Exit function after handling Backspace key
  }

  // Handle Enter key press
  if (keyCode === 13) {
    const output = processCommand(commandBuffer);
    if (isInEditMode()) {
      term.write(`\r\n${output}`);
    } else {
      // If not in edit mode, add a new line and prompt
      const user = getName();
      term.write(`\r\n${output}\r\n${user}$ `);
    }
    commandBuffer = ""; // Reset the command buffer
    return; // Exit function after handling Enter key
  }

  // Handle Ctrl + C key press
  if (domEvent.ctrlKey && domEvent.key === "c") {
    stopMatrix();
    term.write("\r\nInterrupted\r\n$ ");
    commandBuffer = ""; // Reset the command buffer
    return; // Exit function after handling Ctrl+C
  }

  // For regular key presses, append the character to the command buffer and write to terminal
  commandBuffer += key;
  term.write(key);
}

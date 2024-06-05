import { stopMatrix } from "./random.js";
import { loginManager } from "./index.js";
import { isInEditMode } from "./edit.js";
import { isInChatMode, handleChatCommand } from "./chat.js";

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
 * Renders the terminal prompt.
 *
 * @param {Object} term - The xterm.js terminal object.
 */
function renderPrompt(term) {
  const user = loginManager.getUsername();
  const prompt = user ? `${user}$ ` : "$ ";
  term.write(`\r\n${prompt}`);
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

  if (key === "Backspace" || keyCode === 8) {
    if (commandBuffer.length > 0) {
      commandBuffer = commandBuffer.slice(0, -1); // Update the command buffer
      term.write("\b \b"); // Move cursor back, write space to delete char, then move cursor back again
    } else {
      domEvent.preventDefault(); // Prevent the backspace if the command buffer is empty
      console.log("Backspace blocked at prompt");
    }
    return; // Stop further processing
  }

  // Handle Enter key press
  if (keyCode === 13) {
    if (isInChatMode()) {
      handleChatCommand(commandBuffer);
      //renderPrompt(term);
    } else {
      const output = await processCommand(commandBuffer);
      if (isInEditMode()) {
        term.write(`\r\n${output}`);
      } else {
        // If not in edit mode, write the output and render the prompt
        if (output) {
          term.write(`\r\n${output}`);
        }
        renderPrompt(term);
      }
    }
    commandBuffer = "";
    term.scrollToBottom();
    return;
  }

  // Handle Ctrl + C key press
  if (domEvent.ctrlKey && domEvent.key === "c") {
    stopMatrix();
    term.write("\r\nInterrupted");
    commandBuffer = ""; // Reset the command buffer
    renderPrompt(term); // Render the prompt
    return;
  }

  // For regular key presses, append the character to the command buffer and write to terminal
  commandBuffer += key;
  term.write(key);
}

import { term, loginManager, socket } from "../terminal/index.js";
import {
  saveCommandBuffer,
  restoreCommandBuffer,
  updateCommandList,
} from "../terminal/keyInputHandler.js";
import { appendToolToFileData } from "../terminal/fileSystem.js";

export function initializeGame() {
  socket.on("scanResult", (data) => handleGameMessage(data));
  socket.on("hackResult", (data) => handleGameMessage(data));
  socket.on("miningResult", (data) => handleGameMessage(data));
  socket.on("downloadResult", (data) => handleGameMessage(data));
}

function handleGameMessage(data) {
  const state = saveCommandBuffer();
  term.write(`\r\x1b[2K\r`); // Clear the current line and move cursor to the beginning

  const { success, message, error, data: eventData, toolName } = data;

  if (error) {
    term.write(`\r\n${message || "Operation failed"}: ${error}\r\n`);
  } else if (success) {
    term.write(`\r\n${message}\r\n`);
    if (eventData) {
      console.log(eventData);
      term.write(`${formatJSON(eventData)}\r\n`);
    }
    if (toolName) {
      console.log(toolName);
      appendToolToFileData(toolName);
      updateCommandList();
    }
  }

  restoreCommandBuffer(term, state);
}

function formatJSON(data) {
  const jsonString = JSON.stringify(data, null, 2);
  return jsonString.replace(/\n/g, "\r\n").replace(/ /g, "\u00a0");
}

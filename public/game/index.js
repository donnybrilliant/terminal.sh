import { term, loginManager, socket } from "../terminal/index.js";
import {
  saveCommandBuffer,
  restoreCommandBuffer,
} from "../terminal/keyInputHandler.js";

export function initializeGame() {
  socket.on("scanResult", (data) => handleGameMessage(data, "scanResult"));
  socket.on("hackResult", (data) => handleGameMessage(data, "hackResult"));
}

function handleGameMessage(data, eventType) {
  const state = saveCommandBuffer();
  term.write(`\r\x1b[2K\r`); // Clear the current line and move cursor to the beginning

  if (eventType === "scanResult") {
    const { targetIP, details, error } = data;
    if (error) {
      term.write(`\r\nScan failed: ${error}\r\n`);
    } else {
      term.write(`\r\nScan result for ${targetIP}:\r\n`);
      term.write(`${formatJSON(details)}\r\n`);
    }
  } else if (eventType === "hackResult") {
    const { success, targetIP, details, error } = data;
    if (error) {
      term.write(`\r\nHack failed: ${error}\r\n`);
    } else if (success) {
      term.write(`\r\nSuccessfully hacked ${targetIP}:\r\n`);
      term.write(`${formatJSON(details)}\r\n`);
    } else {
      term.write(`\r\nHack attempt failed for ${targetIP}\r\n`);
    }
  }

  restoreCommandBuffer(term, state);
}

function formatJSON(data) {
  const jsonString = JSON.stringify(data, null, 2);
  return jsonString.replace(/\n/g, "\r\n").replace(/ /g, "\u00a0");
}

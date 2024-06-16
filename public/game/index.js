import { term, socket } from "../terminal/index.js";
import {
  saveCommandBuffer,
  restoreCommandBuffer,
  updateCommandList,
} from "../terminal/keyInputHandler.js";
import {
  appendToolToFileData,
  loadTargetFileSystem,
  sshFileData,
} from "../terminal/fileSystem.js";
import { startSSHSession } from "../ssh/index.js";

export function initializeGame() {
  socket.on("scanInternetResult", (data) => handleGameMessage(data));
  socket.on("scanIPResult", (data) => handleGameMessage(data));
  socket.on("hackResult", (data) => handleGameMessage(data));
  socket.on("miningResult", (data) => handleGameMessage(data));
  socket.on("downloadResult", (data) => handleGameMessage(data));
  socket.on("sshExploitResult", (data) => handleGameMessage(data));
  socket.on("passwordSnifferResult", (data) => handleGameMessage(data));
  socket.on("userEnumResult", (data) => handleGameMessage(data));
  socket.on("passwordCrackerResult", (data) => handleGameMessage(data));
  socket.on("sshResult", (data) => handleGameMessage(data));
  socket.on("rootkitResult", (data) => handleGameMessage(data));
}

function handleGameMessage(data) {
  const state = saveCommandBuffer();
  term.write(`\r\x1b[2K\r`); // Clear the current line and move cursor to the beginning

  const {
    success,
    message,
    error,
    data: eventData,
    toolName,
    targetIP,
    ssh,
    load,
  } = data;

  if (error) {
    term.write(`\r\n${message || "Operation failed"}: ${error}\r\n`);
  } else if (success) {
    term.write(`\r\n${message}\r\n`);
    if (eventData) {
      console.log(eventData);
      //term.write(`${formatJSON(eventData)}\r\n`);
      term.write("Check the console for eventData.\r\n");
    }
    if (toolName) {
      appendToolToFileData(toolName);
      updateCommandList();
    }
    if (ssh) {
      startSSHSession(targetIP);
      if (eventData) {
        loadTargetFileSystem(eventData);
      } else {
        loadTargetFileSystem();
      }
    }
    if (load) {
      loadTargetFileSystem(eventData);
    }
  }

  restoreCommandBuffer(term, state);
}

function formatJSON(data) {
  const jsonString = JSON.stringify(data, null, 2);
  return jsonString.replace(/\n/g, "\r\n").replace(/ /g, "\u00a0");
}

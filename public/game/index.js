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
import { startSSHSession, currentSSHSession } from "../ssh/index.js";

export function initializeGame() {
  socket.on("scanInternetResult", (data) => handleGameMessage(data));
  socket.on("scanIPResult", (data) => handleGameMessage(data));
  socket.on("miningResult", (data) => handleGameMessage(data));
  socket.on("miningUpdate", (data) => handleGameMessage(data));
  socket.on("downloadResult", (data) => handleGameMessage(data));
  socket.on("getToolResult", (data) => handleGameMessage(data));
  socket.on("getResult", (data) => handleGameMessage(data));
  socket.on("sshExploitResult", (data) => handleGameMessage(data));
  socket.on("passwordSnifferResult", (data) => handleGameMessage(data));
  socket.on("userEnumResult", (data) => handleGameMessage(data));
  socket.on("passwordCrackerResult", (data) => handleGameMessage(data));
  socket.on("sshResult", (data) => handleGameMessage(data));
  socket.on("rootkitResult", (data) => handleGameMessage(data));
  socket.on("hardwareResult", (data) => handleGameMessage(data));
  socket.on("walletResult", (data) => handleGameMessage(data));
  socket.on("ifconfigResult", (data) => handleGameMessage(data));
  socket.on("exploitedResult", (data) => handleGameMessage(data));
  socket.on("toolsResult", (data) => handleGameMessage(data));
  socket.on("minerResult", (data) => handleGameMessage(data));
  socket.on("userinfoResult", (data) => handleGameMessage(data));
  socket.on("createServerResult", (data) => handleGameMessage(data));
  socket.on("createLocalServerResult", (data) => handleGameMessage(data));
  socket.on("lanSnifferResult", (data) => handleGameMessage(data));
  socket.on("scanLocalNetworkResult", (data) => handleGameMessage(data));
}

function handleGameMessage(data) {
  const state = saveCommandBuffer();
  term.write(`\r\x1b[2K\r`); // Clear the current line and move cursor to the beginning

  const {
    success,
    message,
    error,
    data: eventData,
    tool,
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

    // Rewrite this logic
    if (tool) {
      appendToolToFileData(tool);
      updateCommandList();
    }
    if (ssh) {
      const parentSession = currentSSHSession ? { ...currentSSHSession } : null;
      startSSHSession(targetIP, parentSession);
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

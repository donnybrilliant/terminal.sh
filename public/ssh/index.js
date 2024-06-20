// ssh/index.js
import { term, loginManager, socket } from "../terminal/index.js";
import {
  loadTargetFileSystem,
  loadFileSystem,
  getCurrentDir,
  setCurrentDir,
  getCurrentPath,
  getDirectoryNames,
} from "../terminal/fileSystem.js";
import { getCombinedCommandMap } from "../terminal/commandProcessor.js";

let sshMode = false;
let parentSSHSession = null;
export let currentSSHSession = { targetIP: null, parent: null };

export function startSSHSession(ip) {
  if (currentSSHSession.targetIP) {
    currentSSHSession.parent = currentSSHSession.targetIP;
  }
  currentSSHSession.targetIP = ip;
  console.log(currentSSHSession);
  sshMode = true;
  renderSSHPrompt();
}

export function isInSSHMode() {
  return sshMode;
}

export function handleSSHCommand(command) {
  const [cmd, ...args] = command.split(" ");
  const combinedCommandMap = getCombinedCommandMap(); // Get combined command map

  // this doesnt load the parent ssh filesystem.
  if (command.trim() === ":exit") {
    if (currentSSHSession.parent) {
      currentSSHSession = currentSSHSession.parent;
      // Another message here?
      return "Returning to parent SSH session";
      //renderSSHPrompt();
    } else {
      sshMode = false;
      currentSSHSession = { targetIP: null, parent: null };
      loadFileSystem();
      return "Disconnected from SSH session.";
    }
  }

  const commandFunction = combinedCommandMap[cmd];
  if (commandFunction) {
    return commandFunction(args);
  } else {
    return `Unknown command: ${cmd}`;
  }
}

export function renderSSHPrompt() {
  const user = loginManager.getUsername();
  const prompt = `${user}@${currentSSHSession.targetIP}$ `;
  term.write(prompt);
}

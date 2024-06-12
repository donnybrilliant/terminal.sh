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
export let currentSSHSession = { targetIP: null };

export function startSSHSession(ip) {
  currentSSHSession.targetIP = ip;
  sshMode = true;
  renderSSHPrompt();
}

export function isInSSHMode() {
  return sshMode;
}

export function handleSSHCommand(command) {
  const [cmd, ...args] = command.split(" ");
  const combinedCommandMap = getCombinedCommandMap(); // Get combined command map

  if (command.trim() === ":exit") {
    sshMode = false;
    currentSSHSession.targetIP = null;
    term.write(`\r\nDisconnected from SSH session.\r\n`);
    loadFileSystem(); // Reload the main terminal filesystem
    return;
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

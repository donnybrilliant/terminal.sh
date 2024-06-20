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
export let currentSSHSession = { targetIP: null, parents: [] }; // Allow for multiple parents

export function startSSHSession(ip, parentSession = null) {
  if (currentSSHSession.targetIP) {
    if (parentSession) {
      currentSSHSession.parents.push(parentSession);
    } else {
      currentSSHSession.parents.push({
        targetIP: currentSSHSession.targetIP,
        parents: [...currentSSHSession.parents],
      });
    }
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

  if (command.trim() === ":exit") {
    if (currentSSHSession.parents.length > 0) {
      const parentSession = currentSSHSession.parents.pop();
      currentSSHSession = { ...parentSession };
      loadTargetFileSystem(parentSession.fileSystem);
      return "Returning to parent SSH session";
    } else {
      sshMode = false;
      currentSSHSession = { targetIP: null, parents: [] };
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

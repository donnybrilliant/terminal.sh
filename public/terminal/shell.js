// shell.js
import {
  setCurrentDir,
  getCurrentDir,
  getCurrentPath,
  saveUserHome,
} from "./fileSystem.js";
import { ANSI_COLORS } from "./colors.js";
import { term } from "./index.js";
import { isInSSHMode } from "../ssh/index.js";

/**
 * Mocked ShellJS 'ls' command with list flag support.
 * Lists contents of the given directory, excluding metadata.
 * If '-l' flag is provided, the directory contents are displayed as a list.
 *
 * @param {Array<string>} args - Array containing command arguments. ['-l'] is the supported flag.
 * @returns {string} - Space-separated list of entries in the directory.
 */
function ls(args = []) {
  const path = getCurrentDir(isInSSHMode());
  const listFlag = args.includes("-l");

  const contents = Object.keys(path).filter(
    (key) => key !== "name" && key !== "parent"
  );

  if (contents.length === 0) {
    return isInSSHMode() ? ".........................." : "Directory is empty";
  }

  if (listFlag) {
    return contents
      .map((entry) => {
        const isDir = !path[entry].content;
        const type = isDir
          ? `${ANSI_COLORS.blue}[DIR]${ANSI_COLORS.reset} `
          : `${ANSI_COLORS.green}[FILE]${ANSI_COLORS.reset} `;

        if (isDir) {
          return type + entry;
        } else {
          const url = path[entry].url || "";
          return type + createHyperlink(entry, url);
        }
      })
      .join("\r\n");
  }

  return contents
    .map((entry) => {
      if (!path[entry].content) {
        return entry;
      } else {
        const url = path[entry].url || "";
        return createHyperlink(entry, url);
      }
    })
    .join("  ");
}

/**
 * Wraps a given text with ANSI escape codes to make it appear as a hyperlink in the terminal.
 * When detected by a terminal handler (like xterm-addon-web-links), the text becomes clickable,
 * redirecting the user to the specified URL.
 *
 * @param {string} text - The display text that will appear as a clickable hyperlink in the terminal.
 * @param {string} url - The actual URL to which the hyperlink should point.
 * @returns {string} - The text wrapped with the necessary ANSI escape codes to make it a hyperlink.
 */
function createHyperlink(text, url) {
  return `\x1B]8;;${url}\x1B\\${text}\x1B]8;;\x1B\\`;
}
// for creating links
//createHyperlink(entry, path[entry]);

/**
 * Mocked ShellJS 'cat' command.
 * Displays the contents of a file.
 *
 * @param {string} filename - Name of the file to display.
 * @returns {string} - Content of the file or an error message.
 */

function cat(filename) {
  const currentDir = getCurrentDir(isInSSHMode());
  if (filename in currentDir) {
    const file = currentDir[filename];
    if (file.content) {
      return file.content.replace(/\n/g, "\r\n");
    } else {
      return `cat: ${filename}: Is a directory`;
    }
  } else {
    return `cat: ${filename}: No such file`;
  }
}

/**
 * Mocked ShellJS 'cd' command.
 * Changes the current directory.
 *
 * @param {string} dir - Name of the directory to change into.
 * @returns {string} - Success or error message.
 */
function cd(dir) {
  const result = setCurrentDir(dir, isInSSHMode());
  if (result === true) {
    return pwd();
  } else if (result === "Not a directory") {
    return `cd: ${dir}: Not a directory`;
  } else {
    return `cd: ${dir}: No such directory`;
  }
}

/**
 * Function to handle the 'pwd' command.
 * Shows the current directory path.
 *
 * @returns {string} - The current directory path.
 */
function pwd() {
  return getCurrentPath(isInSSHMode());
}

/**
 * Displays a list of available commands to the user.
 *
 * @returns {string} - A string containing the list of available commands.
 */
function help() {
  return (
    `${ANSI_COLORS.red}Available commands:\r\n` +
    `${ANSI_COLORS.green}ls [-l]${ANSI_COLORS.reset}                  - List directory contents\r\n` +
    `${ANSI_COLORS.green}cat <filename>${ANSI_COLORS.reset}           - Display file contents\r\n` +
    `${ANSI_COLORS.green}edit|vi|nano <filename>${ANSI_COLORS.reset}  - Edit a file\r\n` +
    `${ANSI_COLORS.green}touch <filename>${ANSI_COLORS.reset}         - Create a new file\r\n` +
    `${ANSI_COLORS.green}mkdir <foldername>${ANSI_COLORS.reset}       - Create a new directory\r\n` +
    `${ANSI_COLORS.green}cp <src> <dest>${ANSI_COLORS.reset}          - Copy files/folders\r\n` +
    `${ANSI_COLORS.green}mv <src> <dest>${ANSI_COLORS.reset}          - Move or rename files/folders\r\n` +
    `${ANSI_COLORS.green}rm <filename>${ANSI_COLORS.reset}            - Delete file\r\n` +
    `${ANSI_COLORS.green}rm -r <folder>${ANSI_COLORS.reset}           - Delete folder\r\n` +
    `${ANSI_COLORS.green}cd <directory>${ANSI_COLORS.reset}           - Change current directory\r\n` +
    `${ANSI_COLORS.green}pwd${ANSI_COLORS.reset}                      - Print current directory\r\n` +
    `${ANSI_COLORS.green}info${ANSI_COLORS.reset}                     - Display browser info\r\n` +
    `${ANSI_COLORS.green}name${ANSI_COLORS.reset}                     - Change your username\r\n` +
    `${ANSI_COLORS.green}matrix${ANSI_COLORS.reset}                   - Start the Matrix animation\r\n` +
    `${ANSI_COLORS.green}hack${ANSI_COLORS.reset}                     - Simulate hacking (just for fun!)\r\n` +
    `${ANSI_COLORS.green}chat${ANSI_COLORS.reset}                     - Start Chat\r\n` +
    `${ANSI_COLORS.green}loadtest${ANSI_COLORS.reset}                 - Stolen from xtermjs.org\r\n` +
    `${ANSI_COLORS.green}chars${ANSI_COLORS.reset}                    - Stolen from xtermjs.org\r\n` +
    `${ANSI_COLORS.green}clear${ANSI_COLORS.reset}                    - Clears terminal\r\n` +
    `${ANSI_COLORS.green}login${ANSI_COLORS.reset}                    - Log in or create user\r\n` +
    `${ANSI_COLORS.green}logout${ANSI_COLORS.reset}                   - Log out\r\n` +
    `${ANSI_COLORS.green}help${ANSI_COLORS.reset}                     - Display this help menu`
  );
}

async function remove(args = []) {
  if (args.length !== 2) {
    return "Usage: rm <filename> or rm -r <folder>";
  }

  // Separate out the options (e.g., -r) from the actual target
  const options = args.filter((arg) => arg.startsWith("-"));
  const target = args.find((arg) => !arg.startsWith("-"));

  const currentDir = getCurrentDir(isInSSHMode());

  // If the target is not provided or does not exist in the current directory, return an error
  if (!target || !currentDir[target]) {
    return `${target}: No such file or directory`;
  }

  // Check if the target is a directory
  const isDirectory = typeof currentDir[target] === "object";

  // If it's a directory but -r option is not provided, return an error
  if (isDirectory && !options.includes("-r")) {
    return `${target}: is a directory (use -r to remove directories)`;
  }

  // If everything checks out, delete the target
  delete currentDir[target];
  await saveUserHome();
  return `${target} removed successfully`;
}

function clear() {
  term.clear();
  return "";
}

async function mkdir(dirname) {
  const currentDir = getCurrentDir(isInSSHMode());
  if (currentDir[dirname]) {
    return `mkdir: ${dirname}: File or directory already exists`;
  }
  currentDir[dirname] = {};
  await saveUserHome();
  return `Directory '${dirname}' created`;
}

async function touch(filename) {
  const currentDir = getCurrentDir(isInSSHMode());
  if (currentDir[filename]) {
    return `touch: ${filename}: File already exists`;
  }
  currentDir[filename] = { content: "" };
  await saveUserHome();
  return `File '${filename}' created`;
}

async function cp(source, destination) {
  const currentDir = getCurrentDir(isInSSHMode());

  // Use the "in" operator to check for property existence
  if (!(source in currentDir)) {
    return `cp: ${source}: No such file or directory`;
  }
  if (destination in currentDir) {
    return `cp: ${destination}: File or directory already exists`;
  }
  currentDir[destination] = currentDir[source];
  await saveUserHome();
  return `File '${source}' copied to '${destination}'`;
}

async function mv(source, destination) {
  const currentDir = getCurrentDir(isInSSHMode());

  // Use the "in" operator to check for property existence
  if (!(source in currentDir)) {
    return `mv: ${source}: No such file or directory`;
  }
  if (destination in currentDir) {
    return `mv: ${destination}: File or directory already exists`;
  }
  currentDir[destination] = currentDir[source];
  delete currentDir[source];
  await saveUserHome();
  return `File '${source}' moved to '${destination}'`;
}

export const commands = {
  ls: ls,
  cat: cat,
  cd: cd,
  pwd: pwd,
  help: help,
  rm: remove,
  clear: clear,
  mkdir: mkdir,
  touch: touch,
  cp: cp,
  mv: mv,
};

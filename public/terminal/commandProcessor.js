import { commands } from "./shell.js";
import { term } from "./index.js";
import { loadtest, chars, hack, startMatrix, getClientInfo } from "./random.js";
import {
  editFile,
  saveEdits,
  exitEdit,
  isInEditMode,
  appendToEditedContent,
  getEditedContent,
} from "./edit.js";
import { initializeChat } from "./chat.js";
/**
 * Main function to process terminal commands.
 * Delegates work to individual command functions from the mockShell (shell.js).
 *
 * @param {string} command - The command entered by the user.
 * @returns {string} - The output of the command.
 */
export default function processCommand(command) {
  const [cmd, ...args] = command.split(" ");

  // Check if the system is in edit mode
  if (isInEditMode()) {
    if (cmd.trim() === ":save") {
      return saveEdits(getEditedContent().trim());
    } else if (cmd.trim() === ":exit") {
      return exitEdit();
    } else {
      appendToEditedContent(cmd); // Add the user input to editedContent
      return "";
    }
  }

  // If not in edit mode, proceed with normal command processing
  switch (cmd) {
    case "nano":
    case "vi":
    case "edit":
      return editFile(args[0]);
    case "ls":
      return commands.ls(args);
    case "cd":
      if (args.length !== 1) {
        return "Usage: cd <folder> or ..";
      }
      return commands.cd(args[0]);
    case "cat":
      if (args.length !== 1) {
        return "Usage: cat <filename>";
      }
      return commands.cat(args[0]);

    case "pwd":
      return commands.pwd();
    case "help":
      return commands.help();
    case "loadtest":
      return loadtest(term);
    case "chars":
      return chars(term);
    case "hack":
      return hack(term);
    case "matrix":
      return startMatrix(term);
    case "info":
      return getClientInfo();
    case "name":
      return commands.name(args.join(" "));
    case "rm":
      return commands.rm(args);
    case "clear":
      return commands.clear();
    case "mkdir":
      if (args.length !== 1) {
        return "Usage: mkdir <dirname>";
      }
      return commands.mkdir(args[0]);
    case "touch":
      if (args.length !== 1) {
        return "Usage: touch <filename>";
      }
      return commands.touch(args[0]);
    case "cp":
      if (args.length !== 2) {
        return "Usage: cp <source> <destination>";
      }
      return commands.cp(args[0], args[1]);
    case "mv":
      if (args.length !== 2) {
        return "Usage: mv <source> <destination>";
      }
      return commands.mv(args[0], args[1]);
    case "hola":
      return "hello";
    case "chat":
      initializeChat();
      term.writeln(`Chat initialized. You can start typing messages.`);
      return;

    default:
      return `Unknown command: ${cmd}`;
  }
}

import processCommand from "./commandProcessor.js";
import handleKeyInput from "./keyInputHandler.js";
import ascii from "./ascii.js";
import { populateFileSystem } from "./fileSystem.js";
import { LoginManager } from "./login.js";

export const term = new Terminal({ cursorBlink: true });
export const loginManager = new LoginManager("http://localhost:3000");

document.addEventListener("DOMContentLoaded", function () {
  const terminalContainer = document.getElementById("terminal-container");

  // Create and apply the fit addon
  const fitAddon = new FitAddon.FitAddon();
  term.loadAddon(fitAddon);

  // Create and apply the web links addon
  /*  const webLinksAddon = new WebLinksAddon();
  term.loadAddon(webLinksAddon); */

  // Populate the file system
  populateFileSystem();

  term.open(terminalContainer);
  loginManager.setTerminal(term);
  term.focus();
  fitAddon.fit();

  // Refit on window resize
  window.addEventListener("resize", () => {
    fitAddon.fit();
  });

  // Handle key input
  term.onKey((eventData) => handleKeyInput(eventData, term, processCommand));

  // Print the ascii art
  ascii(term);
});

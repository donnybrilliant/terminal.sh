import processCommand from "./commandProcessor.js";
import handleKeyInput from "./keyInputHandler.js";
import ascii from "./ascii.js";
import { LoginManager } from "../auth/login.js";
import { initializeGame } from "../game/index.js"; // Import initializeGame function

export const term = new Terminal({ cursorBlink: true });
//export const socket = io();
export const socket = io("http://localhost:3000");
export const loginManager = new LoginManager(socket, "http://localhost:3000");

document.addEventListener("DOMContentLoaded", async function () {
  const terminalContainer = document.getElementById("terminal-container");

  // Create and apply the fit addon
  const fitAddon = new FitAddon.FitAddon();
  term.loadAddon(fitAddon);

  // Create and apply the web links addon
  const webLinksAddon = new WebLinksAddon.WebLinksAddon();
  term.loadAddon(webLinksAddon);

  term.open(terminalContainer);
  loginManager.setTerminal(term);
  fitAddon.fit();

  // Refit on window resize
  window.addEventListener("resize", () => {
    fitAddon.fit();
  });

  // Initialize the login state and load filesystem if logged in
  await loginManager.initializeLoginState();
  //loginManager.checkAuthStatus();
  // Initialize the game
  initializeGame();

  // Print the ascii art
  ascii(term);
  // Should ascii be awaited?

  // Handle key input
  term.onKey((eventData) => handleKeyInput(eventData, term, processCommand));

  term.focus();
});

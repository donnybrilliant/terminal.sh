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
  /* 
  console.log(navigator.hardwareConcurrency);
  console.log(navigator.userAgent);
  function bytesToGB(bytes, decimals = 2) {
    const GB = 1024 * 1024 * 1024;
    return (bytes / GB).toFixed(decimals) + " GB";
  }

  // Example usage
  navigator.storage.estimate().then(({ quota }) => {
    console.log(`Quota: ${bytesToGB(quota)}`);
  });

  // no safari
  console.log(navigator);
  console.log(navigator.deviceMemory);
  console.log(navigator.userAgentData);
  //console.log(navigator.userAgentData.platform);
  //console.log(navigator.userAgentData.brands[0].brand);

  // safari
  console.log(navigator.platform);
  console.log(navigator.vendor);
  console.log(navigator.appCodeName, navigator.appName); */
});

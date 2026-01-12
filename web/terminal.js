/**
 * Terminal.sh - xterm.js WebSocket client
 * Optimized for Bubble Tea full-screen rendering
 */

// Get WebSocket URL (ws:// or wss:// based on current page protocol)
function getWebSocketURL() {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const host = window.location.host;
  return `${protocol}//${host}/ws`;
}

// Initialize terminal with scrollback enabled for shell history
const term = new Terminal({
  cursorBlink: false,
  cursorStyle: "block",
  cursorInactiveStyle: "none",
  disableStdin: false,
  scrollback: 10000, // Enable scrollback for command history
  allowProposedApi: true,
  convertEol: false, // We handle line endings from server
  windowsMode: false,
  scrollOnUserInput: true,
  theme: {
    background: "#000000",
    foreground: "#ffffff",
    cursor: "transparent",
    cursorAccent: "transparent",
  },
  fontFamily:
    '"Cascadia Code", "Fira Code", "SF Mono", Menlo, Monaco, "Courier New", monospace',
  fontSize: 14,
  lineHeight: 1.1,
  letterSpacing: 0,
});

// Load addons
const fitAddon = new FitAddon.FitAddon();
const webLinksAddon = new WebLinksAddon.WebLinksAddon();

term.loadAddon(fitAddon);
term.loadAddon(webLinksAddon);

// Open terminal in container
const terminalContainer = document.getElementById("terminal-container");
term.open(terminalContainer);

// Initial fit
fitAddon.fit();

// WebSocket connection
let ws = null;
let pendingResize = null;

// Send resize to server
function sendResize() {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(
      JSON.stringify({
        type: "resize",
        width: term.cols,
        height: term.rows,
      })
    );
  }
}

// Debounced resize handler
let resizeTimeout = null;
function handleResize() {
  fitAddon.fit();

  // Debounce resize messages to server
  if (resizeTimeout) {
    clearTimeout(resizeTimeout);
  }
  resizeTimeout = setTimeout(() => {
    sendResize();
  }, 100);
}

// Handle window resize
window.addEventListener("resize", handleResize);

// Also handle orientation changes on mobile
window.addEventListener("orientationchange", () => {
  setTimeout(handleResize, 100);
});

function connectWebSocket() {
  const wsUrl = getWebSocketURL();
  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    // Clear terminal
    term.reset();
    // Send initial resize
    sendResize();
  };

  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);

      if (message.type === "output") {
        // Write the data directly - server handles all ANSI sequences
        term.write(message.data);
      }
    } catch (e) {
      // If not JSON, write raw data
      term.write(event.data);
    }
  };

  ws.onerror = () => {
    // Connection error - will trigger onclose
  };

  ws.onclose = () => {
    // Show reconnect message
    term.write(
      "\r\n\x1b[31mConnection closed. Reconnecting in 3 seconds...\x1b[0m\r\n"
    );
    setTimeout(connectWebSocket, 3000);
  };
}

// Handle keyboard input
// Send printable characters via onData; keep onKey for control/navigation
term.onData((data) => {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  if (!data) return;
  // Only send printable ASCII characters (avoid control sequences)
  // This prevents double-sending special keys handled in onKey
  if (data.length === 1) {
    const code = data.charCodeAt(0);
    if (code >= 32 && code <= 126) {
      ws.send(
        JSON.stringify({
          type: "input",
          key: data,
          char: data,
          modifiers: [],
        })
      );
    }
  }
});

// Use attachCustomKeyEventHandler to intercept key events BEFORE xterm.js processes them
// This gives us control over the event flow and prevents xterm.js from interfering
term.attachCustomKeyEventHandler((event) => {
  // Handle copy/paste
  const isCopy = (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'c';
  const isPaste = (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'v';
  
  // Handle paste - send as single paste message
  if (isPaste) {
    event.preventDefault();
    if (navigator.clipboard && navigator.clipboard.readText) {
      navigator.clipboard.readText().then(text => {
        if (text && ws && ws.readyState === WebSocket.OPEN) {
          // Send paste as a single message with the full text
          ws.send(JSON.stringify({
            type: 'paste',
            text: text
          }));
        }
      }).catch(err => {
        console.error('Failed to read from clipboard:', err);
      });
    }
    return false; // Prevent xterm.js from processing paste
  }
  
  // Handle copy - Ctrl+C (not Cmd+C on Mac)
  if (isCopy) {
    // Check if it's Ctrl+C (not Cmd+C) - Ctrl is for Windows/Linux, Cmd is for Mac
    const isCtrlC = event.ctrlKey && !event.metaKey && event.key.toLowerCase() === 'c';
    const isCmdC = event.metaKey && !event.ctrlKey && event.key.toLowerCase() === 'c';
    
    if (isCmdC) {
      // Cmd+C on Mac - always allow native copy
      if (term.hasSelection()) {
        return true; // Allow xterm.js default copy behavior
      }
      return true; // Allow native copy even without selection
    }
    
    if (isCtrlC) {
      // Ctrl+C on Windows/Linux
      if (term.hasSelection()) {
        // Text selected - allow copy
        return true; // Allow xterm.js default copy behavior
      }
      // No selection - send Ctrl+C to server to clear input
      event.preventDefault();
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'input',
          key: 'Ctrl+c',
          char: '',
          modifiers: ['Control']
        }));
      }
      return false; // Prevent xterm.js from processing
    }
  }
  
  // For all other keys (including Enter), allow xterm.js to process them
  // This ensures proper event flow and timing
  return true;
});

term.onKey((event) => {
  const { key, domEvent } = event;

  // Prevent browser default for special keys
  if (domEvent.key === "Tab") {
    domEvent.preventDefault();
  }

  // If this is a plain printable character without modifiers, let onData handle it
  if (
    !domEvent.ctrlKey &&
    !domEvent.altKey &&
    !domEvent.metaKey &&
    domEvent.key.length === 1
  ) {
    const code = domEvent.key.charCodeAt(0);
    if (code >= 32 && code <= 126) {
      return;
    }
  }

  // Build modifiers array
  const modifiers = [];
  if (domEvent.ctrlKey) modifiers.push("Control");
  if (domEvent.altKey) modifiers.push("Alt");
  if (domEvent.shiftKey) modifiers.push("Shift");
  if (domEvent.metaKey) modifiers.push("Meta");

  // Determine key name for special keys
  let keyName = key;
  let char = key;

  switch (domEvent.key) {
    case "Enter":
      keyName = "Enter";
      char = "";
      domEvent.preventDefault();
      break;
    case "Backspace":
      keyName = "Backspace";
      char = "";
      break;
    case "Tab":
      keyName = "Tab";
      char = "";
      domEvent.preventDefault();
      break;
    case "Escape":
      keyName = "Esc";
      char = "";
      break;
    case "ArrowUp":
      keyName = domEvent.shiftKey ? "shift+up" : "Up";
      char = "";
      domEvent.preventDefault();
      break;
    case "ArrowDown":
      keyName = domEvent.shiftKey ? "shift+down" : "Down";
      char = "";
      domEvent.preventDefault();
      break;
    case "ArrowLeft":
      keyName = "Left";
      char = "";
      domEvent.preventDefault();
      break;
    case "ArrowRight":
      keyName = "Right";
      char = "";
      domEvent.preventDefault();
      break;
    case "PageUp":
      keyName = "pgup";
      char = "";
      domEvent.preventDefault();
      break;
    case "PageDown":
      keyName = "pgdown";
      char = "";
      domEvent.preventDefault();
      break;
    case "Home":
      keyName = "home";
      char = "";
      break;
    case "End":
      keyName = "end";
      char = "";
      break;
    default:
      // Handle Ctrl+key combinations
      if (domEvent.ctrlKey && !domEvent.metaKey) {
        const ctrlKey = domEvent.key.toLowerCase();
        if (ctrlKey === "c") {
          // Ctrl+C is handled by attachCustomKeyEventHandler
          // It will send to server if no selection, or allow copy if selection exists
          return; // Don't send here, let the handler deal with it
        } else if (ctrlKey === "s") {
          keyName = "Ctrl+s";
          char = "";
          domEvent.preventDefault();
        } else if (ctrlKey === "q") {
          keyName = "Ctrl+q";
          char = "";
          domEvent.preventDefault();
        } else if (ctrlKey === "l") {
          keyName = "Ctrl+l";
          char = "";
          domEvent.preventDefault();
        }
      }
  }

  // Send input message to server
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(
      JSON.stringify({
        type: "input",
        key: keyName,
        char: char,
        modifiers: modifiers,
      })
    );
  }
});

// Handle mouse wheel for in-app scrollback
terminalContainer.addEventListener(
  "wheel",
  (e) => {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    // Determine scroll direction
    const button = e.deltaY < 0 ? "wheelUp" : "wheelDown";

    // Send mouse message to server
    ws.send(
      JSON.stringify({
        type: "mouse",
        button: button,
        x: 0,
        y: 0,
      })
    );

    // Prevent default xterm.js scrolling (we handle it server-side now)
    e.preventDefault();
  },
  { passive: false }
);

// Handle right-click paste
terminalContainer.addEventListener("contextmenu", (e) => {
  // Allow default context menu for copy when text is selected
  if (term.hasSelection()) {
    return; // Let browser show context menu with copy option
  }
  
  // If no selection, handle right-click paste
  e.preventDefault();
  if (navigator.clipboard && navigator.clipboard.readText) {
    navigator.clipboard.readText().then(text => {
      if (text && ws && ws.readyState === WebSocket.OPEN) {
        // Send paste as a single message with the full text
        ws.send(JSON.stringify({
          type: 'paste',
          text: text
        }));
      }
    }).catch(err => {
      console.error('Failed to read from clipboard:', err);
    });
  }
});

// Connect when page loads
document.addEventListener("DOMContentLoaded", () => {
  // Ensure fit happens after layout
  requestAnimationFrame(() => {
    fitAddon.fit();
    connectWebSocket();
    term.focus();
  });
});

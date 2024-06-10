import { loginManager } from "./index.js";
export default function ascii(term) {
  const username = loginManager.getUsername();
  // Welcome message
  const ASCII_TERMINAL = [
    "  _                      _             _       _     ",
    " | |                    (_)           | |     | |    ",
    " | |_ ___ _ __ _ __ ___  _ _ __   __ _| |  ___| |__  ",
    " | __/ _ \\ '__| '_ ` _ \\| | '_ \\ / _` | | / __| '_ \\ ",
    " | ||  __/ |  | | | | | | | | | | (_| | |_\\__ \\ | | |",
    "  \\__\\___|_|  |_| |_| |_|_|_| |_|\\__,_|_(_)___/_| |_|",
  ];

  /*   const ASCII_TERMINAL1 = [
    "▄▄▄█████▓▓█████  ██▀███   ███▄ ▄███▓ ██▓ ███▄    █  ▄▄▄       ██▓    ",
    "▓  ██▒ ▓▒▓█   ▀ ▓██ ▒ ██▒▓██▒▀█▀ ██▒▓██▒ ██ ▀█   █ ▒████▄    ▓██▒    ",
    "▒ ▓██░ ▒░▒███   ▓██ ░▄█ ▒▓██    ▓██░▒██▒▓██  ▀█ ██▒▒██  ▀█▄  ▒██░    ",
    "░ ▓██▓ ░ ▒▓█  ▄ ▒██▀▀█▄  ▒██    ▒██ ░██░▓██▒  ▐▌██▒░██▄▄▄▄██ ▒██░    ",
    "  ▒██▒ ░ ░▒████▒░██▓ ▒██▒▒██▒   ░██▒░██░▒██░   ▓██░ ▓█   ▓██▒░██████▒",
    "  ▒ ░░   ░░ ▒░ ░░ ▒▓ ░▒▓░░ ▒░   ░  ░░▓  ░ ▒░   ▒ ▒  ▒▒   ▓▒█░░ ▒░▓  ░",
    "    ░     ░ ░  ░  ░▒ ░ ▒░░  ░      ░ ▒ ░░ ░░   ░ ▒░  ▒   ▒▒ ░░ ░ ▒  ░",
    "  ░         ░     ░░   ░ ░      ░    ▒ ░   ░   ░ ░   ░   ▒     ░ ░   ",
    "            ░  ░   ░            ░    ░           ░       ░  ░    ░  ░",
    "                                                                     ",
  ]; */

  term.clear();

  const cols = term.cols;
  const startCol = Math.floor((cols - ASCII_TERMINAL[0].length) / 2);

  let completedAnimations = 0;
  const totalCharacters = ASCII_TERMINAL.join("").split(" ").join("").length; // Count of non-space characters

  function dropCharacter(character, x, y, delay, terminal) {
    setTimeout(() => {
      // Erase the original character
      terminal.write("\x1B[" + y + ";" + x + "H ");

      // Check if we're at the bottom of the terminal
      if (y + 1 < term.rows) {
        // Redraw character one position below and continue falling
        terminal.write("\x1B[" + (y + 1) + ";" + x + "H" + character);
        dropCharacter(character, x, y + 1, delay * 0.8, terminal); // Decreased delay for faster fall
      } else {
        completedAnimations++;
        if (completedAnimations === totalCharacters) {
          // All characters have completed animation
          setTimeout(() => {
            term.clear();
            term.write("\x1B[1;1H"); // Reset cursor to top-left
            term.write("Welcome to the terminal.\r\n");
            term.write("Type 'help' to get started.\r\n\r\n");
            term.write(`${username}$ `);
          }, 0); // Reduced post-animation wait time to 800ms
        }
      }
    }, delay);
  }

  // Display the ASCII_TERMINAL at the top
  ASCII_TERMINAL.forEach((line, rowIndex) => {
    term.write("\x1B[" + (1 + rowIndex) + ";" + startCol + "H" + line);
  });

  // Start the dropping animation after 1 second
  setTimeout(() => {
    ASCII_TERMINAL.forEach((line, rowIndex) => {
      for (let columnIndex = 0; columnIndex < line.length; columnIndex++) {
        const character = line[columnIndex];
        if (character !== " ") {
          // We'll skip spaces
          const randomDelay = Math.random() * 250; // Reduced max random delay to 250ms
          dropCharacter(
            character,
            startCol + columnIndex,
            1 + rowIndex,
            randomDelay,
            term
          );
        }
      }
    });
  }, 1000);
}

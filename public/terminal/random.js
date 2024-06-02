import { term } from "./index.js";

export function loadtest(term) {
  // ... rest of the loadtest function ...
  let testData = [];
  let byteCount = 0;
  for (let i = 0; i < 50; i++) {
    let count = 1 + Math.floor(Math.random() * 79);
    byteCount += count + 2;
    let data = new Uint8Array(count + 2);
    data[0] = 0x0a; // \n
    for (let i = 1; i < count + 1; i++) {
      data[i] = 0x61 + Math.floor(Math.random() * (0x7a - 0x61));
    }
    // End each line with \r so the cursor remains constant, this is what ls/tree do and improves
    // performance significantly due to the cursor DOM element not needing to change
    data[data.length - 1] = 0x0d; // \r
    testData.push(data);
  }
  let start = performance.now();
  for (let i = 0; i < 1024; i++) {
    for (const d of testData) {
      term.write(d);
    }
  }
  // Wait for all data to be parsed before evaluating time
  term.write("", () => {
    let isWebglEnabled = false;
    let time = Math.round(performance.now() - start);
    let mbs = ((byteCount / 1024) * (1 / (time / 1000))).toFixed(2);
    term.writeln(
      `\n\r\nWrote ${byteCount}kB in ${time}ms (${mbs}MB/s) using the ${
        isWebglEnabled ? "webgl" : "canvas"
      } renderer`
    );

    term.write("$ ");
  });
}

export function chars(term) {
  // ... rest of the chars function ...
  const _1to8 = [];
  for (let i = 1; i <= 8; i++) {
    _1to8.push(i);
  }
  const _1to16 = [];
  for (let i = 1; i <= 16; i++) {
    _1to16.push(i);
  }
  const _1to24 = [];
  for (let i = 1; i <= 24; i++) {
    _1to24.push(i);
  }
  const _1to32 = [];
  for (let i = 1; i <= 32; i++) {
    _1to32.push(i);
  }
  const _0to35 = [];
  for (let i = 0; i <= 35; i++) {
    _0to35.push(i);
  }
  const _1to64 = [];
  for (let i = 1; i <= 64; i++) {
    _1to64.push(i);
  }
  const _0to255 = [];
  for (let i = 17; i <= 255; i++) {
    _0to255.push(i);
  }
  const lines = [
    ["Ascii ─", "abc123"],
    ["CJK ─", "汉语, 漢語, 日本語, 한국어"],
    [
      "Powerline ─",
      "\ue0b2\ue0b0\ue0b3\ue0b1\ue0b6\ue0b4\ue0b7\ue0b5\ue0ba\ue0b8\ue0bd\ue0b9\ue0be\ue0bc",
    ],
    ["Box drawing ┬", "┌─┬─┐ ┏━┳━┓ ╔═╦═╗ ┌─┲━┓ ╲   ╱"],
    ["            │", "│ │ │ ┃ ┃ ┃ ║ ║ ║ │ ┃ ┃  ╲ ╱"],
    ["            │", "├─┼─┤ ┣━╋━┫ ╠═╬═╣ ├─╄━┩   ╳"],
    ["            │", "│ │ │ ┃ ┃ ┃ ║ ║ ║ │ │ │  ╱ ╲"],
    ["            └", "└─┴─┘ ┗━┻━┛ ╚═╩═╝ └─┴─┘ ╱   ╲"],
    ["Block elem ─", "░▒▓█ ▁▂▃▄▅▆▇█ ▏▎▍▌▋▊▉"],
    ["Emoji ─", "😉 👋"],
    [
      "16 color ─",
      [
        ..._1to8.map((e) => `\x1b[3${e - 1}m●`),
        ..._1to8.map((e) => `\x1b[1;3${e - 1}m●`),
      ].join(""),
    ],
    [
      "256 color ┬",
      [..._0to35.map((e) => `\x1b[38;5;${16 + 36 * 0 + e}m●`)].join(""),
    ],
    [
      "          │",
      [..._0to35.map((e) => `\x1b[38;5;${16 + 36 * 1 + e}m●`)].join(""),
    ],
    [
      "          │",
      [..._0to35.map((e) => `\x1b[38;5;${16 + 36 * 2 + e}m●`)].join(""),
    ],
    [
      "          │",
      [..._0to35.map((e) => `\x1b[38;5;${16 + 36 * 3 + e}m●`)].join(""),
    ],
    [
      "          │",
      [..._0to35.map((e) => `\x1b[38;5;${16 + 36 * 4 + e}m●`)].join(""),
    ],
    [
      "          │",
      [..._0to35.map((e) => `\x1b[38;5;${16 + 36 * 5 + e}m●`)].join(""),
    ],
    [
      "          └",
      [..._1to24.map((e) => `\x1b[38;5;${232 + e - 1}m●`)].join(""),
    ],
    [
      "True color ┬",
      [..._1to64.map((e) => `\x1b[38;2;${64 * 0 + e - 1};0;0m●`)].join(""),
    ],
    [
      "           │",
      [..._1to64.map((e) => `\x1b[38;2;${64 * 1 + e - 1};0;0m●`)].join(""),
    ],
    [
      "           │",
      [..._1to64.map((e) => `\x1b[38;2;${64 * 2 + e - 1};0;0m●`)].join(""),
    ],
    [
      "           └",
      [..._1to64.map((e) => `\x1b[38;2;${64 * 3 + e - 1};0;0m●`)].join(""),
    ],
    [
      "Styles ─",
      [
        "\x1b[1mBold",
        "\x1b[2mFaint",
        "\x1b[3mItalics",
        "\x1b[7mInverse",
        "\x1b[9mStrikethrough",
        "\x1b[8mInvisible",
      ].join("\x1b[0m, "),
    ],
    [
      "Underlines ─",
      [
        "\x1b[4:1mStraight",
        "\x1b[4:2mDouble",
        "\x1b[4:3mCurly",
        "\x1b[4:4mDotted",
        "\x1b[4:5mDashed",
      ].join("\x1b[0m, "),
    ],
  ];
  const maxLength = lines.reduce((p, c) => Math.max(p, c[0].length), 0);
  term.write("\r\n");
  term.writeln(
    lines.map((e) => `${e[0].padStart(maxLength)}  ${e[1]}\x1b[0m`).join("\r\n")
  );
  return "";
}

async function writeWithDelay(term, message, delay) {
  return new Promise((resolve) => {
    setTimeout(() => {
      term.writeln(message);
      resolve();
    }, delay);
  });
}

export async function hack(term) {
  await writeWithDelay(term, "Initiating connection to remote node...", 500);
  await writeWithDelay(term, "Connection established.", 600);
  await writeWithDelay(term, "Decrypting node access...", 900);
  await writeWithDelay(term, "Decryption successful.", 1100);
  await writeWithDelay(term, "Accessing mainframe...", 800);
  await writeWithDelay(term, "Running exploit scripts...", 1200);
  await writeWithDelay(term, "Exploit successful! Node compromised.", 1000);
  await writeWithDelay(term, "Gathering data...", 1500);
  await writeWithDelay(term, "Data downloaded. Disconnecting...", 700);
  term.writeln("Disconnected from remote node. Operation successful.");
  term.write("$ ");
  return "";
}

let matrixInterval = null; // Module-scoped variable to hold the interval ID.

export function startMatrix(term) {
  const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
  const columns = term.cols; // Assume 'term' has 'cols' property to tell terminal width
  const drops = [];

  // Initialize drop positions
  for (let i = 0; i < columns; i++) {
    drops[i] = 1;
  }

  function drawMatrixRain() {
    // Draw each drop
    for (let i = 0; i < drops.length; i++) {
      const text = characters.charAt(
        Math.floor(Math.random() * characters.length)
      );
      term.write(`\x1b[${drops[i]};${i + 1}H\x1b[32m${text}\x1b[0m`);

      // Random chance of resetting drop
      if (Math.random() > 0.975) {
        drops[i] = 0;
      }

      drops[i]++;
    }
  }

  // Start the matrix rain effect
  matrixInterval = setInterval(drawMatrixRain, 100);
  return "";
}

export function stopMatrix() {
  if (matrixInterval) {
    clearInterval(matrixInterval);
    matrixInterval = null;
  }
  term.write("\x1b[2J\x1b[H"); // Clear screen and reset cursor position
}

export function getClientInfo() {
  return [
    `User Agent: ${navigator.userAgent}`,
    `Screen Dimensions: ${screen.width}x${screen.height}`,
    `Window Dimensions: ${window.innerWidth}x${window.innerHeight}`,
    `Current URL: ${window.location.href}`,
    `Referrer URL: ${document.referrer}`,
  ].join("\r\n");
}

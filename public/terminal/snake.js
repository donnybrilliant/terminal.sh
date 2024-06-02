import { term } from "./index.js";
let snake;
// Top of snake.js
let isActive = false;
let interval; // Declare this so that it can be accessed throughout the module

const SNAKE_SIZE = 1; // You can adjust this based on your xterm cell size
const DIRECTIONS = {
  UP: { x: 0, y: -SNAKE_SIZE },
  DOWN: { x: 0, y: SNAKE_SIZE },
  LEFT: { x: -SNAKE_SIZE, y: 0 },
  RIGHT: { x: SNAKE_SIZE, y: 0 },
};
let currentDirection = DIRECTIONS.RIGHT;

function initGame() {
  snake = [
    { x: 10, y: 10 },
    { x: 9, y: 10 },
    { x: 8, y: 10 },
  ];
  spawnFood();
}

let foods = []; // Multiple foods instead of a single food item
const MAX_FOOD_COUNT = 3; // Set to how many pieces of food you want

function spawnFood() {
  const cols = term.cols;
  const rows = term.rows;

  while (foods.length < MAX_FOOD_COUNT) {
    foods.push({
      x: Math.floor(Math.random() * cols) * SNAKE_SIZE,
      y: Math.floor(Math.random() * rows) * SNAKE_SIZE,
    });
  }
}

function gameLoop() {
  const cols = term.cols;
  const rows = term.rows;
  // Move the snake
  const head = Object.assign({}, snake[0]);
  head.x += currentDirection.x;
  head.y += currentDirection.y;
  snake.unshift(head);

  let foodEaten = false;

  // In gameLoop function, loop over foods to check if snake eats any of them
  for (let i = 0; i < foods.length; i++) {
    if (snake[0].x === foods[i].x && snake[0].y === foods[i].y) {
      foods.splice(i, 1); // Remove the eaten food item
      spawnFood(); // This will fill up food if it's less than MAX_FOOD_COUNT
      foodEaten = true; // Mark that food was eaten on this loop iteration
      break;
    }
  }

  // Only pop the tail off if no food was eaten
  if (!foodEaten) {
    snake.pop();
  }

  // Check for collisions
  for (let i = 1; i < snake.length; i++) {
    if (snake[i].x === snake[0].x && snake[i].y === snake[0].y) {
      // Game over
      return false;
    }
  }

  if (
    snake[0].x < 2 ||
    snake[0].x > cols || // Use > instead of >=
    snake[0].y < 2 ||
    snake[0].y > rows // Use > instead of >=
  ) {
    return false; // Game over when hitting the edge
  }

  return true; // Game continues
}

function renderGame(terminal) {
  terminal.clear();

  // Draw snake
  for (const segment of snake) {
    terminal.write("\x1B[" + segment.y + ";" + segment.x + "H*");
  }

  // Draw food
  // In renderGame function, loop over foods to render them
  foods.forEach((foodItem) => {
    terminal.write("\x1B[" + foodItem.y + ";" + foodItem.x + "H#");
  });
}

function handleInput(data) {
  switch (data) {
    case "\u001B[A": // UP arrow
      if (currentDirection !== DIRECTIONS.DOWN)
        currentDirection = DIRECTIONS.UP;
      break;
    case "\u001B[B": // DOWN arrow
      if (currentDirection !== DIRECTIONS.UP)
        currentDirection = DIRECTIONS.DOWN;
      break;
    case "\u001B[C": // RIGHT arrow
      if (currentDirection !== DIRECTIONS.LEFT)
        currentDirection = DIRECTIONS.RIGHT;
      break;
    case "\u001B[D": // LEFT arrow
      if (currentDirection !== DIRECTIONS.RIGHT)
        currentDirection = DIRECTIONS.LEFT;
      break;
  }
}

let inputDisposer;

function startSnakeGame(terminal) {
  initGame();
  isActive = true;
  interval = setInterval(() => {
    if (!gameLoop()) {
      clearInterval(interval);

      // ASCII Art for "Game Over"
      const gameOverAscii = [
        "  _____                         ____                 ",
        " / ____|                       / __ \\                ",
        "| |  __  __ _ _ __ ___   ___  | |  | |_   _____ _ __ ",
        "| | |_ |/ _` | '_ ` _ \\ / _ \\ | |  | \\ \\ / / _ \\ '__|",
        "| |__| | (_| | | | | | |  __/ | |__| |\\ V /  __/ |   ",
        " \\_____|\\__,_|_| |_| |_|\\___|  \\____/  \\_/ \\___|_|   ",
      ];

      // Position the cursor in the middle of the screen
      const cols = term.cols;
      const rows = term.rows;
      const middleRow =
        Math.floor(rows / 2) - Math.floor(gameOverAscii.length / 2);
      const startCol = Math.floor((cols - gameOverAscii[0].length) / 2);
      for (let i = 0; i < gameOverAscii.length; i++) {
        term.write(
          "\x1B[" + (middleRow + i) + ";" + startCol + "H" + gameOverAscii[i]
        );
      }

      if (inputDisposer) {
        inputDisposer.dispose(); // detach the input listener
      }
    } else {
      renderGame(terminal);
    }
  }, 100); // adjust the interval for game speed

  // Listen for input and capture the disposer
  inputDisposer = terminal.onData(handleInput);
}

// New function to stop the snake game
function stopSnakeGame() {
  if (isActive) {
    clearInterval(interval);
    isActive = false; // Set the game as no longer active
  }
}

export {
  initGame,
  gameLoop,
  renderGame,
  handleInput,
  startSnakeGame,
  stopSnakeGame,
};

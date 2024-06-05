import { logAction } from "./utils/logger.js";
import {
  readJSONFile,
  writeJSONFile,
  USERS_FILE_PATH,
} from "./utils/fileUtils.js";
import fs from "fs";

let users = [];
let onlineUsers = new Set(); // Use a set to track online users
let guestCount = 0; // Counter for Guest users

// Load users from JSON file at startup
async function loadUsers() {
  try {
    users = await readJSONFile(USERS_FILE_PATH);
  } catch (err) {
    console.error("Error loading users:", err);
    users = [];
  }
}

// Log messages to individual JSON files for each room
async function logMessage(room, message) {
  const filePath = `./data/messages/${room}.json`;
  let messages = [];
  try {
    if (fs.existsSync(filePath)) {
      messages = await readJSONFile(filePath);
    }
    messages.push(message);
    await writeJSONFile(filePath, messages);
  } catch (err) {
    console.error(`Error logging message to ${filePath}:`, err);
  }
}

export async function setupSocket(io) {
  await loadUsers();

  const chatNamespace = io.of("/chat");

  chatNamespace.on("connection", (socket) => {
    let username;
    let currentRoom = "general"; // Default room is general

    socket.on("joinGeneral", async (providedUsername) => {
      if (username) return; // Prevent double counting if joinGeneral is called multiple times

      username = providedUsername;
      socket.username = username; // Store username in socket for later use
      socket.currentRoom = "general"; // Set initial room to general
      socket.join("general");

      if (username === "Guest") {
        guestCount++;
      } else {
        onlineUsers.add(username);
      }

      logAction(username, "Joined general chat");
      socket.broadcast
        .to("general")
        .emit("message", `${username} has joined the chat`);
    });

    socket.on("chatMessage", async (data) => {
      const { message, username } = data;
      const room = socket.currentRoom; // Get current room of the user
      logAction(username, `Message in ${room}: ${message}`);
      await logMessage(room, { username, message });
      chatNamespace.to(room).emit("message", `${username}: ${message}`);
    });

    socket.on("createAlliance", async (data) => {
      let { usernames, creator } = data;
      const allianceRoom = `alliance-${creator}-${Date.now()}`;

      // Ensure the creator is included in the alliance
      if (!usernames.includes(creator)) {
        usernames.push(creator);
      }

      // Load the current users from the file again to ensure data consistency
      let currentUsers = [];
      try {
        currentUsers = await readJSONFile(USERS_FILE_PATH);
      } catch (err) {
        console.error("Error loading users:", err);
      }

      // Update users to include the alliance room
      for (const username of usernames) {
        const user = currentUsers.find((u) => u.username === username);
        if (user) {
          if (!user.alliance) {
            user.alliance = [];
          }
          user.alliance.push(allianceRoom);
        }
      }

      // Save updated users with alliances
      await writeJSONFile(USERS_FILE_PATH, currentUsers);

      // Create an empty file for the new alliance room
      await writeJSONFile(`./data/messages/${allianceRoom}.json`, []);

      usernames.forEach((user) => {
        const userSocket = findUserSocket(user, chatNamespace);
        if (userSocket) {
          userSocket.join(allianceRoom);
        }
      });

      logAction(creator, `Created alliance with ${usernames.join(", ")}`);
      chatNamespace
        .to(allianceRoom)
        .emit("message", `Alliance created by ${creator}`);

      // Automatically join the creator to the new alliance room
      const creatorSocket = findUserSocket(creator, chatNamespace);
      if (creatorSocket) {
        creatorSocket.leave("general");
        creatorSocket.join(allianceRoom);
        creatorSocket.currentRoom = allianceRoom; // Update the current room
        creatorSocket.emit(
          "message",
          `You have been moved to the new alliance room: ${allianceRoom}`
        );
      }
    });

    socket.on("disconnect", () => {
      if (!username) return; // Prevent decrementing if disconnect is called before joinGeneral

      if (username === "Guest") {
        guestCount--;
      } else {
        onlineUsers.delete(username);
      }

      socket.broadcast
        .to("general")
        .emit("message", `${username} has left the chat`);
      logAction(username, "Disconnected");
    });

    socket.on("exit", (username) => {
      if (username === "Guest") {
        guestCount--;
      } else {
        onlineUsers.delete(username);
      }

      socket.broadcast
        .to("general")
        .emit("message", `${username} has left the chat`);
      logAction(username, "Exited chat");
    });

    socket.on("listUsers", async () => {
      let usersList = await readJSONFile(USERS_FILE_PATH);
      usersList = usersList.map((user) =>
        onlineUsers.has(user.username) ? `${user.username} *` : user.username
      );

      if (guestCount > 0) {
        usersList.unshift(`Guest (${guestCount} online)`); // Add Guest count at the beginning of the list
      }

      socket.emit("userList", usersList);
    });
  });

  io.on("connection", (socket) => {
    // General and game-related socket handling can be added here
  });
}

function findUserSocket(username, namespace) {
  const sockets = namespace.sockets;
  for (let [id, socket] of sockets) {
    if (socket.username === username) {
      return socket;
    }
  }
  return null;
}

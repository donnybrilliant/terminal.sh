import { logAction } from "./utils/logger.js";
import {
  readJSONFile,
  writeJSONFile,
  CHAT_ROOMS_FILE_PATH,
  USERS_FILE_PATH,
} from "./utils/fileUtils.js";
import fs from "fs";

let chatRooms = {
  general: [],
};
let users = [];
let onlineUsers = new Set(); // Use a set to track online users
let guestCount = 0; // Counter for Guest users

// Load chat rooms and users from JSON files at startup
async function loadChatRooms() {
  try {
    chatRooms = await readJSONFile(CHAT_ROOMS_FILE_PATH);
  } catch (err) {
    console.error("Error loading chat rooms:", err);
    chatRooms = { general: [] }; // Initialize with an empty general room if there's an error
  }
}

async function loadUsers() {
  try {
    users = await readJSONFile(USERS_FILE_PATH);
  } catch (err) {
    console.error("Error loading users:", err);
    users = [];
  }
}

// Save chat rooms to JSON file
async function saveChatRooms() {
  try {
    await writeJSONFile(CHAT_ROOMS_FILE_PATH, chatRooms);
  } catch (err) {
    console.error("Error saving chat rooms:", err);
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
  await loadChatRooms();
  await loadUsers();

  const chatNamespace = io.of("/chat");

  chatNamespace.on("connection", (socket) => {
    let username;

    socket.on("joinGeneral", async (providedUsername) => {
      if (username) return; // Prevent double counting if joinGeneral is called multiple times

      username = providedUsername;
      socket.username = username; // Store username in socket for later use
      socket.join("general");

      if (!chatRooms.general.includes(username)) {
        chatRooms.general.push(username);
        await saveChatRooms();
      }

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
      const { room, message, username } = data;
      logAction(username, `Message in ${room}: ${message}`);
      await logMessage(room, { username, message });
      chatNamespace.to(room).emit("message", `${username}: ${message}`);
    });

    socket.on("createAlliance", async (data) => {
      const { usernames, creator } = data;
      const allianceRoom = `alliance-${creator}-${Date.now()}`;
      chatRooms[allianceRoom] = usernames;
      usernames.forEach((user) => {
        const userSocket = findUserSocket(user, chatNamespace);
        if (userSocket) {
          userSocket.join(allianceRoom);
        }
      });
      await saveChatRooms();
      logAction(creator, `Created alliance with ${usernames.join(", ")}`);
      chatNamespace
        .to(allianceRoom)
        .emit("message", `Alliance created by ${creator}`);
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
        usersList.push(`Guest (${guestCount} online)`);
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

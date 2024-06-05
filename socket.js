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
    socket.on("joinGeneral", async (username) => {
      socket.join("general");
      if (!chatRooms.general.includes(username)) {
        chatRooms.general.push(username);
        await saveChatRooms();
      }
      //if (!users.includes(username)) {
      if (!users.some((user) => user.username === username)) {
        users.push({ username, online: true });
        await writeJSONFile(USERS_FILE_PATH, users); // Save only at login
      } else {
        const user = users.find((u) => u.username === username);
        if (user) user.online = true;
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
      const username = socket.username;
      if (username) {
        const user = users.find((u) => u.username === username);
        if (user) user.online = false;
        socket.broadcast
          .to("general")
          .emit("message", `${username} has left the chat`);
        logAction(username, "Disconnected");
      }
    });

    socket.on("exit", (username) => {
      const user = users.find((u) => u.username === username);
      if (user) user.online = false;
      socket.broadcast
        .to("general")
        .emit("message", `${username} has left the chat`);
      logAction(username, "Exited chat");
    });

    socket.on("listUsers", () => {
      const onlineUsers = users.filter((u) => u.online).map((u) => u.username);
      socket.emit("userList", onlineUsers);
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

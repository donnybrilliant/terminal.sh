import { logAction } from "./utils/logger.js";
import {
  readJSONFile,
  writeJSONFile,
  CHAT_ROOMS_FILE_PATH,
} from "./utils/fileUtils.js";

let chatRooms = {
  general: [],
};

// Load chat rooms from JSON file at startup
async function loadChatRooms() {
  try {
    chatRooms = await readJSONFile(CHAT_ROOMS_FILE_PATH);
  } catch (err) {
    console.error("Error loading chat rooms:", err);
    // If there's an error (e.g., file doesn't exist), initialize with an empty general room
    chatRooms = { general: [] };
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

export async function setupSocket(io) {
  await loadChatRooms(); // Load chat rooms when setting up sockets

  io.on("connection", (socket) => {
    socket.on("joinGeneral", async (username) => {
      socket.join("general");
      if (!chatRooms.general.includes(username)) {
        chatRooms.general.push(username);
      }
      await saveChatRooms();
      logAction(username, "Joined general chat");
      io.to("general").emit("message", `${username} has joined the chat\r\n`);
    });

    socket.on("chatMessage", async (data) => {
      const { room, message, username } = data;
      logAction(username, `Message in ${room}: ${message}`);
      io.to(room).emit("message", `${username}: ${message}\r\n`);
    });

    socket.on("createAlliance", async (data) => {
      const { usernames, creator } = data;
      const allianceRoom = `alliance-${creator}-${Date.now()}`;
      chatRooms[allianceRoom] = usernames;
      usernames.forEach((user) => {
        const userSocket = findUserSocket(user, io);
        if (userSocket) {
          userSocket.join(allianceRoom);
        }
      });
      await saveChatRooms();
      logAction(creator, `Created alliance with ${usernames.join(", ")}`);
      io.to(allianceRoom).emit("message", `Alliance created by ${creator}\r\n`);
    });

    socket.on("disconnect", async () => {
      console.log("A user disconnected");
      // Optional: Remove user from chatRooms if needed
    });
  });
}

function findUserSocket(username, io) {
  const sockets = io.sockets.sockets;
  for (let [id, socket] of sockets) {
    if (socket.username === username) {
      return socket;
    }
  }
  return null;
}

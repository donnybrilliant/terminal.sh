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
    chatRooms = { general: [] }; // Initialize with an empty general room if there's an error
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
  await loadChatRooms();

  const chatNamespace = io.of("/chat");

  chatNamespace.on("connection", (socket) => {
    socket.on("joinGeneral", async (username) => {
      socket.join("general");
      if (!chatRooms.general.includes(username)) {
        chatRooms.general.push(username);
      }
      await saveChatRooms();
      logAction(username, "Joined general chat");
      chatNamespace
        .to("general")
        .emit("message", `${username} has joined the chat`);
    });

    socket.on("chatMessage", async (data) => {
      const { room, message, username } = data;
      logAction(username, `Message in ${room}: ${message}`);
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
      console.log("A user disconnected");
      // Optional: Remove user from chatRooms if needed
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

// index.js
import { Server } from "socket.io";
import { setupChatHandlers } from "./chatHandlers.js";
import { setupAllianceHandlers } from "./allianceHandlers.js";
import { logAction } from "../utils/logger.js";
import {
  getUsers,
  setUsers,
  getOnlineUsers,
  getGuestCount,
  incrementGuestCount,
  decrementGuestCount,
  addOnlineUser,
  removeOnlineUser,
  findUserSocket,
  saveUsers,
} from "../utils/userUtils.js";

export async function setupSocket(io) {
  await getUsers();
  const chatNamespace = io.of("/chat");

  chatNamespace.on("connection", (socket) => {
    let username;
    socket.currentRoom = "general";

    socket.on("joinGeneral", async (providedUsername) => {
      if (username) return;

      username = providedUsername;
      socket.username = username;
      socket.join("general");

      if (username === "Guest") {
        incrementGuestCount();
      } else {
        addOnlineUser(username);
      }

      logAction(username, "Joined general chat");
      socket.broadcast
        .to("general")
        .emit("message", `${username} has joined the chat`);
    });

    setupChatHandlers(socket, chatNamespace);
    setupAllianceHandlers(socket, chatNamespace);

    const handleDisconnectOrExit = async () => {
      if (!username) return;

      if (username === "Guest") {
        decrementGuestCount();
      } else {
        removeOnlineUser(username);
      }

      socket.broadcast
        .to("general")
        .emit("message", `${username} has left the chat`);
      logAction(username, "Disconnected");
    };

    socket.on("disconnect", handleDisconnectOrExit);
    socket.on("exit", handleDisconnectOrExit);

    socket.on("listUsers", async () => {
      const users = await getUsers();
      let usersList = users.map((user) =>
        getOnlineUsers().has(user.username)
          ? `${user.username} *`
          : user.username
      );

      if (getGuestCount() > 0) {
        usersList.unshift(`Guest (${getGuestCount()} online)`);
      }

      socket.emit("userList", usersList);
    });
  });

  io.on("connection", (socket) => {
    // General and game-related socket handling can be added here
  });
}

// sockets/index.js
import passport from "passport";
import { setupAuthHandlers } from "./authHandlers.js";
import { setupFileSystemHandlers } from "./fileSystemHandlers.js";
import { setupChatHandlers } from "./chatHandlers.js";
import { setupAllianceHandlers } from "./allianceHandlers.js";
import { setupGameHandlers } from "./gameHandlers.js";
import { logAction } from "../utils/logger.js";
import {
  getUsers,
  addOnlineUser,
  removeOnlineUser,
  incrementGuestCount,
  decrementGuestCount,
} from "../utils/userUtils.js";

function onlyForHandshake(middleware) {
  return (req, res, next) => {
    // Apply middleware only for new handshakes (no session ID)
    const isHandshake = req._query.sid === undefined;
    if (isHandshake) {
      middleware(req, res, next);
    } else {
      next();
    }
  };
}

export function setupSocket(io, sessionMiddleware) {
  io.engine.use(onlyForHandshake(sessionMiddleware));
  io.engine.use(onlyForHandshake(passport.session()));

  io.on("connection", (socket) => {
    let username = "guest";

    logAction(username, "User connected");

    setupAuthHandlers(socket, io);
    setupFileSystemHandlers(socket, io);
    setupGameHandlers(socket, io);

    socket.on("disconnect", () => {
      logAction(username, "User disconnected");
    });
  });

  const chatNamespace = io.of("/chat");

  chatNamespace.on("connection", (socket) => {
    let username;
    let hasDisconnected = false;
    socket.currentRoom = "general";

    socket.on("joinGeneral", async (providedUsername) => {
      if (username) return;

      username = String(providedUsername); // Ensure username is a string
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

    const handleDisconnectOrExit = async (reason) => {
      if (!username || hasDisconnected) return;
      hasDisconnected = true;

      const currentRoom = socket.currentRoom || "general";
      if (currentRoom !== "general") {
        socket.leave(currentRoom);
        socket.broadcast
          .to(currentRoom)
          .emit("message", `${username} has left the room.`);
        socket.join("general");
      }
      socket.broadcast
        .to("general")
        .emit("message", `${username} has left the chat`);

      if (username === "Guest") {
        decrementGuestCount();
      } else {
        removeOnlineUser(username);
      }

      logAction(username, `Disconnected from chat: ${reason}`);
      socket.disconnect(true); // Disconnect the socket
    };

    socket.on("disconnect", (reason) => handleDisconnectOrExit(reason));
    socket.on("exit", () => handleDisconnectOrExit("exit"));

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
}

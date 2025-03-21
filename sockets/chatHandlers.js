import { logMessage, logAction } from "../utils/logger.js";
import { getUsers } from "../utils/userUtils.js";

export function setupChatHandlers(socket, chatNamespace) {
  socket.on("chatMessage", async (data) => {
    const { message, username } = data;
    const room = socket.currentRoom;
    logAction(username, `Message in ${room}: ${message}`);
    await logMessage(room, { username, message });
    chatNamespace.to(room).emit("message", `${username}: ${message}`);
  });

  socket.on("joinRoom", async (room) => {
    const users = await getUsers();
    const user = users.find((u) => u.username === socket.username);

    if (user && user.alliance && user.alliance.includes(room)) {
      if (socket.currentRoom) {
        const oldRoom = socket.currentRoom;
        socket.leave(oldRoom);
        if (oldRoom !== "general") {
          socket.broadcast
            .to(oldRoom)
            .emit("message", `${socket.username} has left the room.`);
        }
      }
      socket.join(room);
      socket.currentRoom = room;
      socket.emit("message", `You have joined the room: ${room}`);
      socket.broadcast
        .to(room)
        .emit("message", `${socket.username} has joined the room.`);
      socket.emit("roomChanged", room);
    } else {
      socket.emit("message", "You do not have access to this room.");
    }
  });

  socket.on("leaveRoom", () => {
    if (socket.currentRoom) {
      const room = socket.currentRoom;
      socket.leave(room);
      socket.broadcast
        .to(room)
        .emit("message", `${socket.username} has left the room.`);
      socket.join("general");
      socket.currentRoom = "general";
      socket.emit(
        "message",
        "You have left the room and joined the general room."
      );
      socket.emit("roomChanged", "general");
    }
  });
}

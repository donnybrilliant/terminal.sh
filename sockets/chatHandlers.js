import { logMessage, logAction } from "../utils/logger.js";

export function setupChatHandlers(socket, chatNamespace) {
  socket.on("chatMessage", async (data) => {
    const { message, username } = data;
    const room = socket.currentRoom;
    logAction(username, `Message in ${room}: ${message}`);
    await logMessage(room, { username, message });
    chatNamespace.to(room).emit("message", `${username}: ${message}`);
  });

  socket.on("joinRoom", (room) => {
    if (socket.currentRoom) {
      const oldRoom = socket.currentRoom;
      socket.leave(oldRoom);
      // Notify others in the old room that the user has left, except if the old room is "general"
      if (oldRoom !== "general") {
        socket.broadcast
          .to(oldRoom)
          .emit("message", `${socket.username} has left the room.`);
      }
    }
    socket.join(room);
    socket.currentRoom = room;
    socket.emit("message", `You have joined the room: ${room}`);
    // Notify others in the new room that the user has joined
    socket.broadcast
      .to(room)
      .emit("message", `${socket.username} has joined the room.`);
  });

  socket.on("leaveRoom", () => {
    if (socket.currentRoom) {
      const room = socket.currentRoom;
      socket.leave(room);
      // Notify others in the room that the user has left
      socket.broadcast
        .to(room)
        .emit("message", `${socket.username} has left the room.`);
      socket.join("general");
      socket.currentRoom = "general";
      socket.emit(
        "message",
        "You have left the room and joined the general room."
      );
    }
  });
}

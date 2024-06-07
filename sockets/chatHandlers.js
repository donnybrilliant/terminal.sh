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
    const previousRoom = socket.currentRoom;
    if (previousRoom) {
      socket.leave(previousRoom);
      chatNamespace
        .to(previousRoom)
        .emit("message", `${socket.username} has left the chat`);
    }
    socket.join(room);
    socket.currentRoom = room;
    chatNamespace
      .to(room)
      .emit("message", `${socket.username} has joined the chat`);
    socket.emit("message", `You have joined the room: ${room}`);
  });

  socket.on("leaveRoom", () => {
    const previousRoom = socket.currentRoom;
    if (previousRoom && previousRoom !== "general") {
      socket.leave(previousRoom);
      chatNamespace
        .to(previousRoom)
        .emit("message", `${socket.username} has left the chat`);
      socket.currentRoom = "general";
      socket.join("general");
      socket.emit(
        "message",
        "You have left the current room and joined the general room."
      );
    }
  });
}

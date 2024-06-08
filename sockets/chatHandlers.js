// chatHandlers.js
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
      socket.leave(socket.currentRoom);
    }
    socket.join(room);
    socket.currentRoom = room;
    socket.emit("message", `You have joined the room: ${room}`);
  });

  socket.on("leaveRoom", () => {
    console.log(socket.currentRoom);
    if (socket.currentRoom) {
      socket.leave(socket.currentRoom);
      socket.join("general");
      socket.currentRoom = "general";
      socket.emit(
        "message",
        "You have left the current room and joined the general room."
      );
    }
  });
}

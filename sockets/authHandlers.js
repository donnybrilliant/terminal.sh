import jwt from "jsonwebtoken";
import {
  readJSONFile,
  writeJSONFile,
  USERS_FILE_PATH,
} from "../utils/fileUtils.js";

// should be in a .env? how does that work serving static like this?
const JWT_SECRET = "your_jwt_secret"; // Use a strong secret key in production

export function setupAuthHandlers(socket) {
  // Authenticate user after initial connection
  socket.on("authenticate", (token, callback) => {
    jwt.verify(token, JWT_SECRET, (err, decoded) => {
      if (err) {
        return callback({ success: false, message: "Authentication failed" });
      }
      socket.user = decoded;
      callback({ success: true, user: socket.user });
    });
  });

  socket.on("check-auth", () => {
    socket.emit("auth-status", {
      authenticated: !!socket.user,
      user: socket.user,
    });
  });

  socket.on("setName", async (data, callback) => {
    const { oldName, newName } = data;
    try {
      const users = await readJSONFile(USERS_FILE_PATH);
      const user = users.find((u) => u.username === oldName);

      if (!user) {
        callback({ success: false, message: "User not found" });
        return;
      }

      if (users.some((u) => u.username === newName)) {
        callback({ success: false, message: "Username already exists" });
        return;
      }

      // Update username in users
      user.username = newName;

      await writeJSONFile(USERS_FILE_PATH, users);

      callback({
        success: true,
        message: `Name updated to ${newName}`,
        user: { username: newName },
      });
    } catch (error) {
      console.error(`Error updating username: ${error.message}`);
      callback({ success: false, message: error.message });
    }
  });
}

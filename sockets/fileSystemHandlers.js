// fileSystemHandlers.js
import {
  readJSONFile,
  writeJSONFile,
  FILE_SYSTEM_PATH,
  USERS_FILE_PATH,
  INTERNET_FILE_PATH,
} from "../utils/fileUtils.js";

export function setupFileSystemHandlers(socket, io) {
  socket.on("loadFileSystem", async (callback) => {
    try {
      let baseFileSystem = await readJSONFile(FILE_SYSTEM_PATH);
      let users = await readJSONFile(USERS_FILE_PATH);

      // Dynamically add user directories to the filesystem
      users.forEach((user) => {
        baseFileSystem.root.home.users[user.username] = {
          ip: user.ip,
        };
      });

      if (socket.user) {
        const user = users.find((u) => u.id === socket.user.id);
        if (user) {
          const userHomeData = {
            ...user.home,
            README: "Welcome, " + user.username,
          };
          baseFileSystem.root.home.users[user.username] = userHomeData;
        }
      } else {
        if (!baseFileSystem.root.home.users.guest) {
          baseFileSystem.root.home.users.guest = {
            README:
              "You are not logged in. There should be some information here...",
          };
        }
      }

      callback({ success: true, data: baseFileSystem });
    } catch (error) {
      callback({
        success: false,
        message: `Error loading filesystem: ${error.message}`,
      });
    }
  });

  socket.on("loadTargetFileSystem", async ({ targetIP }, callback) => {
    try {
      const internet = await readJSONFile(INTERNET_FILE_PATH);
      const target = internet[targetIP];

      if (!target) {
        return callback({
          success: false,
          message: "Target IP not found",
        });
      }

      callback({
        success: true,
        data: target.filesystem,
      });
    } catch (error) {
      callback({
        success: false,
        message: `Error loading target filesystem: ${error.message}`,
      });
    }
  });

  socket.on("saveUserHome", async (homeData, callback) => {
    const username = socket.user ? socket.user.username : null;
    if (!username) {
      return callback({ success: false, message: "User not authenticated" });
    }

    try {
      let users = await readJSONFile(USERS_FILE_PATH);
      const userIndex = users.findIndex((u) => u.username === username);
      if (userIndex === -1) {
        callback({ success: false, message: "User not found" });
        return;
      }

      users[userIndex].home = homeData;
      await writeJSONFile(USERS_FILE_PATH, users);
      callback({ success: true, message: "User home updated successfully" });
    } catch (error) {
      callback({
        success: false,
        message: `Error saving user home: ${error.message}`,
      });
    }
  });
}

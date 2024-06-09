// fileSystemHandlers.js
import {
  readJSONFile,
  writeJSONFile,
  FILE_SYSTEM_PATH,
  USERS_FILE_PATH,
} from "../utils/fileUtils.js";

export function setupFileSystemHandlers(socket, io) {
  socket.on("loadFileSystem", async (callback) => {
    try {
      let baseFileSystem = await readJSONFile(FILE_SYSTEM_PATH);
      let userHomeData = {};

      if (socket.user) {
        let users = await readJSONFile(USERS_FILE_PATH);
        const user = users.find((u) => u.id === socket.user.id);

        // Convert users array to object if necessary
        if (Array.isArray(baseFileSystem.root.home.users)) {
          baseFileSystem.root.home.users =
            baseFileSystem.root.home.users.reduce((acc, username) => {
              acc[username] = { README: "User directory for " + username };
              return acc;
            }, {});
        }

        // Merge user-specific home data
        userHomeData = {
          ...user.home,
          README: "Welcome, " + user.username,
        };
        baseFileSystem.root.home.users[user.username] = userHomeData;
      } else {
        // Guest setup
        if (!baseFileSystem.root.home.users.guest) {
          baseFileSystem.root.home.users.guest = {
            README: "You are not logged in.",
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

  /*   socket.on("set-name", async (data, callback) => {
    const { oldName, newName } = data;
    try {
      const users = await readJSONFile(USERS_FILE_PATH);
      const fileSystem = await readJSONFile(FILE_SYSTEM_PATH);
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

      // Update username in filesystem
      if (fileSystem.root.home.users[oldName]) {
        fileSystem.root.home.users[newName] = {
          ...fileSystem.root.home.users[oldName],
          README: "Welcome, " + newName,
        };
        delete fileSystem.root.home.users[oldName];
        await writeJSONFile(FILE_SYSTEM_PATH, fileSystem);
      }

      callback({
        success: true,
        message: `Name updated to ${newName}`,
        user: { username: newName },
      });
    } catch (error) {
      callback({ success: false, message: error.message });
    }
  }); */

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

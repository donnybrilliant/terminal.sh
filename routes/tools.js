import express from "express";
import { sendResponse } from "../utils/responseUtils.js";
import errorHandler from "../utils/errorHandler.js";
import {
  readJSONFile,
  writeJSONFile,
  USERS_FILE_PATH,
  FILE_SYSTEM_PATH,
} from "../utils/fileUtils.js";

const router = express.Router();

router.get("/filesystem", async (req, res, next) => {
  try {
    let baseFileSystem = await readJSONFile(FILE_SYSTEM_PATH);

    if (req.isAuthenticated()) {
      let users = await readJSONFile(USERS_FILE_PATH);
      const user = users.find((u) => u.id === req.user.id);

      // Assuming baseFileSystem is structured such that `baseFileSystem.root.home.users` is an array
      // Convert it to an object for easier manipulation
      if (Array.isArray(baseFileSystem.root.home.users)) {
        baseFileSystem.root.home.users = baseFileSystem.root.home.users.reduce(
          (acc, username) => {
            acc[username] = { README: "User directory for " + username };
            return acc;
          },
          {}
        );
      }

      // Merge user-specific directory
      baseFileSystem.root.home.users[user.username] = {
        ...user.home,
        README: "Welcome, " + user.username,
      };

      sendResponse(res, 200, baseFileSystem);
    } else {
      sendResponse(res, 200, baseFileSystem);
    }
  } catch (err) {
    next(err);
  }
});

router.post("/set-name", async (req, res, next) => {
  const { oldName, newName } = req.body;
  try {
    let users = await readJSONFile(USERS_FILE_PATH);
    let fileSystem = await readJSONFile(FILE_SYSTEM_PATH);

    let user = users.find((u) => u.username === oldName);
    if (!user) {
      return sendResponse(res, 400, {}, "User not found.");
    }

    if (users.some((u) => u.username === newName)) {
      return sendResponse(res, 400, {}, "Username already exists.");
    }

    const userIndex = fileSystem.root.home.users.indexOf(oldName);
    if (userIndex === -1) {
      return sendResponse(
        res,
        404,
        {},
        "User directory not found in file system."
      );
    }

    fileSystem.root.home.users[userIndex] = newName; // Update the username in the array
    user.username = newName;
    //delete fileSystem.root.home.users[oldName]; // Delete the old username from the object

    await writeJSONFile(USERS_FILE_PATH, users);
    await writeJSONFile(FILE_SYSTEM_PATH, fileSystem);

    sendResponse(res, 200, {}, `Name updated to ${newName}`);
  } catch (err) {
    next(err);
  }
});

router.post("/update-user-home", async (req, res, next) => {
  if (!req.isAuthenticated()) {
    return sendResponse(res, 403, {}, "Not authenticated");
  }

  try {
    let users = await readJSONFile(USERS_FILE_PATH);
    const userIndex = users.findIndex((u) => u.id === req.user.id);
    if (userIndex === -1) {
      return sendResponse(res, 400, {}, "User not found");
    }

    users[userIndex].home = req.body.home;
    await writeJSONFile(USERS_FILE_PATH, users);

    sendResponse(res, 200, {}, "User home updated successfully");
  } catch (err) {
    next(err);
  }
});

// Error handling middleware should be at the end of all routes
router.use(errorHandler);

export default router;

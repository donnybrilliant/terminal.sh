import express from "express";
import fs from "fs";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
const router = express.Router();

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const DATA_DIR = path.join(__dirname, "../data");
const USERS_FILE_PATH = path.join(DATA_DIR, "users.json");
const FILE_SYSTEM_PATH = path.join(DATA_DIR, "filesystem.json");

router.get("/filesystem", (req, res) => {
  if (req.isAuthenticated()) {
    let users = JSON.parse(fs.readFileSync(USERS_FILE_PATH, "utf-8"));
    const user = users.find((u) => u.id === req.user.id);
    res.json(user.home);
  } else {
    let fileSystem = JSON.parse(fs.readFileSync(FILE_SYSTEM_PATH, "utf-8"));
    res.json(fileSystem);
  }
});

router.post("/set-name", (req, res) => {
  const { oldName, newName } = req.body;
  let users = JSON.parse(fs.readFileSync(USERS_FILE_PATH, "utf-8"));
  let user = users.find((u) => u.username === oldName);

  if (!user) {
    return res.status(400).json({ success: false, message: "User not found." });
  }

  if (users.find((u) => u.username === newName)) {
    return res
      .status(400)
      .json({ success: false, message: "Username already exists." });
  }

  user.username = newName;
  fs.writeFileSync(USERS_FILE_PATH, JSON.stringify(users, null, 2));

  let fileSystem = JSON.parse(fs.readFileSync(FILE_SYSTEM_PATH, "utf-8"));
  fileSystem.root.home.users[newName] = fileSystem.root.home.users[oldName];
  delete fileSystem.root.home.users[oldName];
  fs.writeFileSync(FILE_SYSTEM_PATH, JSON.stringify(fileSystem, null, 2));

  res.json({ success: true, message: `Name updated to ${newName}` });
});

router.post("/update-user-home", (req, res) => {
  if (req.isAuthenticated()) {
    let users = JSON.parse(fs.readFileSync(USERS_FILE_PATH, "utf-8"));
    const userIndex = users.findIndex((u) => u.id === req.user.id);
    if (userIndex !== -1) {
      users[userIndex].home = req.body.home;
      fs.writeFileSync(USERS_FILE_PATH, JSON.stringify(users, null, 2));
      res.json({ success: true });
    } else {
      res.status(400).json({ success: false, message: "User not found" });
    }
  } else {
    res.status(403).json({ success: false, message: "Not authenticated" });
  }
});

export default router;

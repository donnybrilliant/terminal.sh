import express from "express";
import http from "http";
import { Server } from "socket.io";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
import passport from "./auth.js";
import session from "express-session";
import fs from "fs";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();
const server = http.createServer(app);
const io = new Server(server);

const admin = io.of("/admin");

app.use(express.json()); // This is equivalent to bodyParser.json() if using Express 4.16.0+
app.use(express.static(path.join(__dirname, "public")));
app.use(express.static(__dirname + "/node_modules/@xterm/"));

app.use(
  session({
    secret: "keyboard cat",
    resave: false,
    saveUninitialized: false,
    store: new session.MemoryStore(), // Use MemoryStore for development
  })
);

app.use(passport.authenticate("session"));

app.post("/login", function (req, res, next) {
  passport.authenticate("local", function (err, user, info) {
    if (err) {
      return next(err);
    }
    if (!user) {
      return res.status(400).json({
        success: false,
        message: info.message || "Authentication failed",
      });
    }
    req.logIn(user, function (err) {
      if (err) {
        return next(err);
      }
      return res.json({
        success: true,
        message: "Authentication succeeded",
        user: user,
      });
    });
  })(req, res, next);
});

app.post("/logout", function (req, res) {
  req.logout(function (err) {
    if (err) {
      console.log(err);
      return res.status(500).json({ message: "Logout failed", error: err });
    }
    res.json({ message: "Logged out successfully" });
  });
});

app.get("/filesystem", (req, res) => {
  const fsData = JSON.parse(
    fs.readFileSync(path.join(__dirname, "data/filesystem.json"), "utf-8")
  );
  res.json(fsData);
});

app.post("/set-name", (req, res) => {
  const { oldName, newName } = req.body;
  const usersFilePath = path.join(__dirname, "data/users.json");
  const fileSystemPath = path.join(__dirname, "data/filesystem.json");

  let users = JSON.parse(fs.readFileSync(usersFilePath, "utf-8"));
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
  fs.writeFileSync(usersFilePath, JSON.stringify(users, null, 2));

  let fileSystem = JSON.parse(fs.readFileSync(fileSystemPath, "utf-8"));
  fileSystem.root.home.users[newName] = fileSystem.root.home.users[oldName];
  delete fileSystem.root.home.users[oldName];
  fs.writeFileSync(fileSystemPath, JSON.stringify(fileSystem, null, 2));

  res.json({ success: true, message: `Name updated to ${newName}` });
});

app.get("/", (req, res) => {
  res.sendFile(path.join(__dirname, "public", "index.html"));
});

app.get("/auth-status", (req, res) => {
  if (req.isAuthenticated()) {
    res.json({ authenticated: true, user: req.user });
  } else {
    res.json({ authenticated: false });
  }
});

server.listen(3000, () => {
  console.log("listening on localhost:3000");
});

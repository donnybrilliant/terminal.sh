import express from "express";
import http from "http";
import { Server } from "socket.io";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
import bodyParser from "body-parser";
import fs from "fs";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();
const server = http.createServer(app);
const io = new Server(server);

const jsonParser = bodyParser.json();

const admin = io.of("/admin");

app.set("views", path.join(__dirname, "views"));
app.set("view engine", "ejs");

app.use(express.static(path.join(__dirname, "public")));
app.use(express.static(__dirname + "/node_modules/@xterm/"));

app.get("/", (req, res) => {
  const room = "default-room";
  let messages = [];
  if (fs.existsSync(`./data/messages/${room}.json`)) {
    messages = JSON.parse(
      fs.readFileSync(`./data/messages/${room}.json`, "utf-8")
    );
  }
  res.render("index", { room: room, messages: messages });
});

admin.on("connection", (socket) => {
  socket.on("join", (data) => {
    socket.join(data.room);

    // Load existing messages
    const roomMessagesPath = `./data/messages/${data.room}.json`;
    let messages = [];
    if (fs.existsSync(roomMessagesPath)) {
      messages = JSON.parse(fs.readFileSync(roomMessagesPath, "utf-8"));
    }

    // Send existing messages to the newly connected client
    socket.emit("load messages", messages);

    const msg =
      data.username != null
        ? `${data.username} joined the room!`
        : "New user joined the room!";
    admin.in(data.room).emit("chat message", msg);
  });

  socket.on("chat message", (data) => {
    let messages = [];
    const roomMessagesPath = `./data/messages/${data.room}.json`;
    if (fs.existsSync(roomMessagesPath)) {
      messages = JSON.parse(fs.readFileSync(roomMessagesPath, "utf-8"));
    }
    messages.push({ username: data.username, msg: data.msg });
    fs.writeFileSync(roomMessagesPath, JSON.stringify(messages));
    admin.in(data.room).emit("chat message", `${data.username}: ${data.msg}`);
  });

  socket.on("disconnect", () => {
    admin.emit("chat message", `User disconnected`);
  });
});

server.listen(3000, () => {
  console.log("listening on localhost:3000");
});

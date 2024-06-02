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

let rooms = JSON.parse(fs.readFileSync("./data/rooms.json", "utf-8"));

app.set("views", path.join(__dirname, "views"));
app.set("view engine", "ejs");

app.use(express.static(path.join(__dirname, "public")));

app.get("/", (req, res) => {
  res.render(__dirname + "/index.ejs", { rooms: rooms });
});

app.get("/:room", (req, res) => {
  const room = req.params.room;
  let messages = [];
  if (fs.existsSync(`./data/messages/${room}.json`)) {
    messages = JSON.parse(
      fs.readFileSync(`./data/messages/${room}.json`, "utf-8")
    );
  }
  res.render(__dirname + "/room.ejs", { room: room, messages: messages });
});

app.post("/newroom", jsonParser, (req, res) => {
  const room = req.body.room;
  app.get("/" + room, (req, res) => {
    res.render(__dirname + "/room.ejs", { room: room });
  });
  if (!rooms.includes(req.body.room)) {
    rooms.push(room);
    if (req.body.save) {
      let rooms = JSON.parse(fs.readFileSync("./data/rooms.json", "utf-8"));
      const newRooms = rooms.concat([req.body.room]);
      fs.writeFileSync("./data/rooms.json", JSON.stringify(newRooms));
    }
    res.send({
      room: room,
    });
  } else {
    res.send({
      error: "room already exist",
    });
  }
});

admin.on("connection", (socket) => {
  socket.on("join", (data) => {
    socket.join(data.room);
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

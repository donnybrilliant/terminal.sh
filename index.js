import express from "express";
import http from "http";
import { Server } from "socket.io";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
import passport from "./auth.js";
import session from "express-session";
import authRoutes from "./routes/auth.js";
import toolRoutes from "./routes/tools.js";
import morgan from "morgan";
import errorHandler from "./utils/errorHandler.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();
const server = http.createServer(app);
const io = new Server(server);

const admin = io.of("/admin");

app.use(express.json());
app.use(express.static(path.join(__dirname, "public")));
app.use(express.static(__dirname + "/node_modules/@xterm/"));

app.use(
  session({
    secret: "keyboard cat",
    resave: false,
    saveUninitialized: false,
    store: new session.MemoryStore(),
  })
);

app.use(passport.authenticate("session"));

app.use(authRoutes);
app.use(toolRoutes);

app.get("/", (req, res) => {
  res.sendFile(path.join(__dirname, "public", "index.html"));
});

app.use(morgan("combined"));
app.use(errorHandler);

server.listen(3000, () => {
  console.log("listening on localhost:3000");
});

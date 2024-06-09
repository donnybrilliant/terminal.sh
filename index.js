import express from "express";
import http from "http";
import { Server } from "socket.io";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
import passport from "./auth.js";
import authRoutes from "./routes/auth.js"; // Assuming you have a routes/auth.js file
import morgan from "morgan";
import errorHandler from "./utils/errorHandler.js";
import { setupSocket } from "./sockets/index.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();
const server = http.createServer(app);
const io = new Server(server);

app.use(passport.initialize());
app.use(express.json());
app.use(express.static(path.join(__dirname, "public")));
app.use(express.static(__dirname + "/node_modules/@xterm/"));

app.use(morgan("dev"));
app.use(authRoutes);
app.use(errorHandler);

app.get("/", (req, res) => {
  res.sendFile(path.join(__dirname, "public", "index.html"));
});

// Initialize the socket setup
setupSocket(io);

server.listen(3000, () => {
  console.log("listening on localhost:3000");
});

// main server file (e.g., index.js)
import express from "express";
import http from "http";
import { Server } from "socket.io";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
import passport from "passport";
import bodyParser from "body-parser";
import session from "express-session";
import authRoutes from "./routes/auth.js";
import morgan from "morgan";
import errorHandler from "./utils/errorHandler.js";
import { setupSocket } from "./sockets/index.js";
import cookieParser from "cookie-parser";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();
const server = http.createServer(app);
const io = new Server(server);

const sessionMiddleware = session({
  secret: "keyboard cat",
  resave: false,
  saveUninitialized: false,
  store: new session.MemoryStore(),
});

app.use(express.json());
// If using body-parser or similar, ensure it's configured here.
//app.use(bodyParser.urlencoded({ extended: false }));
//app.use(express.urlencoded({ extended: false }));
app.use(sessionMiddleware);
//app.use(passport.initialize());
app.use(passport.session());
app.use(express.static(path.join(__dirname, "public")));
app.use(express.static(__dirname + "/node_modules/@xterm/"));

app.use(authRoutes);
app.get("/", (req, res) => {
  res.sendFile(path.join(__dirname, "public", "index.html"));
});

app.use((req, res, next) => {
  console.log("Session Data:", req.session);
  console.log("User Data:", req.user); // This will be set after authentication
  next();
});

app.use(morgan("combined"));
app.use(errorHandler);

setupSocket(io, sessionMiddleware);

server.listen(3000, () => {
  console.log("listening on localhost:3000");
});

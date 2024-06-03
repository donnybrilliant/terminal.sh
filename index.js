import express from "express";
import http from "http";
import { Server } from "socket.io";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
import passport from "./auth.js";
import session from "express-session";

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

app.use(passport.initialize());
app.use(passport.session());

app.post("/login", function (req, res, next) {
  if (req.isAuthenticated()) {
    return res
      .status(400)
      .json({ success: false, message: "User already logged in" });
  }

  passport.authenticate("local", function (err, user, info) {
    if (err) {
      return next(err);
    }
    if (!user) {
      return res.status(400).json({ success: false, message: info.message });
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

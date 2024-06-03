import express from "express";
import http from "http";
import { Server } from "socket.io";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
import fs from "fs";
import passport from "passport";

import LocalStrategy from "passport-local";
import session from "express-session";
import JsonStore from "express-session-json";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();
const server = http.createServer(app);
const io = new Server(server);

const admin = io.of("/admin");

app.use(express.json()); // This is equivalent to bodyParser.json() if using Express 4.16.0+
app.use(express.static(path.join(__dirname, "public")));
app.use(express.static(__dirname + "/node_modules/@xterm/"));

// Using MemoryStore for development purposes
const sessionStore = new session.MemoryStore();

app.use(
  session({
    secret: "keyboard cat",
    resave: false,
    saveUninitialized: false,
    store: new session.MemoryStore(), // Use MemoryStore for development
  })
);

app.use(passport.authenticate("session"));
app.use((req, res, next) => {
  res.locals.isAuthenticated = req.isAuthenticated();
  next();
});

passport.serializeUser(function (user, cb) {
  process.nextTick(function () {
    cb(null, { id: user.id, username: user.username });
  });
});

passport.deserializeUser(function (user, cb) {
  process.nextTick(function () {
    return cb(null, user);
  });
});

passport.use(
  new LocalStrategy(function (username, password, done) {
    console.log("Username:", username, "Password:", password); // Debugging output
    let usersArray = JSON.parse(
      fs.readFileSync(path.join(__dirname, "data/users.json"))
    );
    let user = usersArray.find((u) => u.username === username); // Fix the comparison issue here
    if (user && user.password === password) {
      return done(null, user);
    } else {
      return done(null, false);
    }
  })
);

//app.post("/login", passport.authenticate("local"));

app.post("/login", function (req, res, next) {
  passport.authenticate("local", function (err, user, info) {
    if (err) {
      return next(err);
    }
    if (!user) {
      return res
        .status(400)
        .json({ success: false, message: "Authentication failed" });
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
  req.logout();
  res.json({ message: "Logged out" });
});

app.get("/", (req, res) => {
  res.sendFile(path.join(__dirname, "public", "index.html"));
});

server.listen(3000, () => {
  console.log("listening on localhost:3000");
});

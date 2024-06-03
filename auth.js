import fs from "fs";
import path from "path";
import passport from "passport";
import LocalStrategy from "passport-local";
import bcrypt from "bcryptjs";
import { fileURLToPath } from "url";
import { dirname } from "path";
import { v4 as uuidv4 } from "uuid"; // Import uuid library

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const DATA_DIR = path.join(__dirname, "data");
const USERS_FILE_PATH = path.join(DATA_DIR, "users.json");
const FILE_SYSTEM_PATH = path.join(DATA_DIR, "filesystem.json");

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

function generateUniqueIP(users) {
  const usedIPs = new Set(users.map((user) => user.ip));
  let ip;
  do {
    ip = Array(4)
      .fill(0)
      .map(() => Math.floor(Math.random() * 256))
      .join(".");
  } while (usedIPs.has(ip));
  return ip;
}

passport.use(
  new LocalStrategy(async (username, password, done) => {
    try {
      let users = JSON.parse(fs.readFileSync(USERS_FILE_PATH, "utf-8"));
      let user = users.find((u) => u.username === username);

      if (user) {
        // User exists, check password
        const match = await bcrypt.compare(password, user.password);
        if (match) {
          return done(null, user);
        } else {
          return done(null, false, { message: "Incorrect password." });
        }
      } else {
        // No user found, create new user
        const hashedPassword = await bcrypt.hash(password, 10);
        user = {
          id: uuidv4(), // Generate a unique ID for the user
          username,
          password: hashedPassword,
          ip: generateUniqueIP(users),
          home: {},
        };
        users.push(user);
        fs.writeFileSync(USERS_FILE_PATH, JSON.stringify(users, null, 2));

        // Update filesystem.json
        let fileSystem = JSON.parse(fs.readFileSync(FILE_SYSTEM_PATH, "utf-8"));
        fileSystem.root.home.users.push(username);
        fs.writeFileSync(FILE_SYSTEM_PATH, JSON.stringify(fileSystem, null, 2));

        return done(null, user);
      }
    } catch (error) {
      return done(error);
    }
  })
);

export default passport;

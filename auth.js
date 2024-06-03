// auth.js
import fs from "fs";
import path from "path";
import passport from "passport";
import LocalStrategy from "passport-local";
import bcrypt from "bcryptjs"; // Assuming you will hash passwords for security
import { fileURLToPath } from "url";
import { dirname } from "path";

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
          id: users.length + 1,
          username,
          password: hashedPassword,
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

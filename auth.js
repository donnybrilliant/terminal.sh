// auth.js
import passport from "passport";
import LocalStrategy from "passport-local";
import bcrypt from "bcryptjs";
import { v4 as uuidv4 } from "uuid"; // Import uuid library
import {
  readJSONFile,
  writeJSONFile,
  USERS_FILE_PATH,
  FILE_SYSTEM_PATH,
} from "./utils/fileUtils.js"; // Import the utility functions

passport.serializeUser(function (user, cb) {
  process.nextTick(function () {
    cb(null, user);
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
      let users = await readJSONFile(USERS_FILE_PATH);
      let user = users.find((u) => u.username === username);

      if (user) {
        // User exists, check password
        const match = await bcrypt.compare(password, user.password);
        if (match) {
          console.log("User authenticated:", user);
          return done(null, user);
        } else {
          return done(null, false, { message: "Incorrect password." });
        }
      } else {
        const lowercaseUsername = username.toLowerCase();
        if (
          lowercaseUsername === "admin" ||
          lowercaseUsername === "user" ||
          lowercaseUsername === "guest"
        ) {
          return done(null, false, {
            message: "Username cannot be 'admin', 'user', or 'guest'.",
          });
        }
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
        await writeJSONFile(USERS_FILE_PATH, users);

        /*         // Update filesystem.json
        let fileSystem = await readJSONFile(FILE_SYSTEM_PATH);
        fileSystem.root.home.users.push(username);
        await writeJSONFile(FILE_SYSTEM_PATH, fileSystem); */

        console.log("New user created:", user);
        return done(null, user);
      }
    } catch (error) {
      return done(error);
    }
  })
);

export default passport;

import passport from "passport";
import { Strategy as LocalStrategy } from "passport-local";
import { Strategy as JwtStrategy, ExtractJwt } from "passport-jwt";
import bcrypt from "bcryptjs";
import jwt from "jsonwebtoken";
import { v4 as uuidv4 } from "uuid";
import {
  readJSONFile,
  writeJSONFile,
  USERS_FILE_PATH,
  FILE_SYSTEM_PATH,
} from "./utils/fileUtils.js";

const JWT_SECRET = "your_jwt_secret"; // Use a strong secret key in production

passport.serializeUser((user, cb) => {
  cb(null, user.id);
});

passport.deserializeUser(async (id, cb) => {
  const users = await readJSONFile(USERS_FILE_PATH);
  const user = users.find((u) => u.id === id);
  cb(null, user);
});

// should be in a util
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
          id: uuidv4(),
          username,
          password: hashedPassword,
          ip: generateUniqueIP(users),
          home: {},
        };
        users.push(user);

        // Add user to filesystem.json
        let fileSystem = await readJSONFile(FILE_SYSTEM_PATH);
        fileSystem.root.home.users.push(username);

        // Write changes to both USERS_FILE_PATH and FILE_SYSTEM_PATH
        await Promise.all([
          writeJSONFile(USERS_FILE_PATH, users),
          writeJSONFile(FILE_SYSTEM_PATH, fileSystem),
        ]);

        return done(null, user);
      }
    } catch (error) {
      return done(error);
    }
  })
);

passport.use(
  new JwtStrategy(
    {
      jwtFromRequest: ExtractJwt.fromAuthHeaderAsBearerToken(),
      secretOrKey: JWT_SECRET,
    },
    async (payload, done) => {
      try {
        const users = await readJSONFile(USERS_FILE_PATH);
        const user = users.find((u) => u.id === payload.id);
        if (user) {
          return done(null, user);
        } else {
          return done(null, false);
        }
      } catch (err) {
        return done(err);
      }
    }
  )
);

export const generateToken = (user) => {
  return jwt.sign({ id: user.id, username: user.username }, JWT_SECRET, {
    expiresIn: "1h",
  });
};

export default passport;

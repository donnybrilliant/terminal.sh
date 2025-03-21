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
  INTERNET_FILE_PATH,
} from "./utils/fileUtils.js";
import {
  generateUniqueIP,
  generateLocalIP,
  generateUniqueMAC,
} from "./utils/ipUtils.js";

const JWT_SECRET = "your_jwt_secret"; // Use a strong secret key in production

passport.serializeUser((user, cb) => {
  cb(null, user.id);
});

passport.deserializeUser(async (id, cb) => {
  const users = await readJSONFile(USERS_FILE_PATH);
  const user = users.find((u) => u.id === id);
  cb(null, user);
});

passport.use(
  new LocalStrategy(async (username, password, done) => {
    try {
      let users = await readJSONFile(USERS_FILE_PATH);
      let internet = await readJSONFile(INTERNET_FILE_PATH);
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
        if (lowercaseUsername === "guest") {
          return done(null, false, {
            message: "Username cannot be 'guest'.",
          });
        }

        // No user found, create new user
        const hashedPassword = await bcrypt.hash(password, 10);
        const externalIP = generateUniqueIP(users, internet);
        const localIP = generateLocalIP();
        const mac = generateUniqueMAC(users, internet);

        user = {
          id: uuidv4(),
          username,
          password: hashedPassword,
          ip: externalIP,
          localIP,
          mac,
          home: {},
          level: 0,
          experience: 0,
          resources: {
            cpu: 200,
            bandwidth: 300,
            ram: 24,
          },
          wallet: {
            crypto: 15,
            data: 1200,
          },
          tools: [],
          achievements: [],
          inventory: {
            items: [],
            currency: 500,
          },
          localNetwork: {},
        };
        users.push(user);

        await writeJSONFile(USERS_FILE_PATH, users);

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

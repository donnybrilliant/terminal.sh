import jwt from "jsonwebtoken";

// should be in a .env? how does that work serving static like this?
const JWT_SECRET = "your_jwt_secret"; // Use a strong secret key in production

export function setupAuthHandlers(socket) {
  // Authenticate user after initial connection
  socket.on("authenticate", (token, callback) => {
    jwt.verify(token, JWT_SECRET, (err, decoded) => {
      if (err) {
        return callback({ success: false, message: "Authentication failed" });
      }
      socket.user = decoded;
      callback({ success: true, user: socket.user });
      console.log("User authenticated:", socket.user);
    });
  });

  socket.on("check-auth", () => {
    socket.emit("auth-status", {
      authenticated: !!socket.user,
      user: socket.user,
    });
  });
}

import {
  readJSONFile,
  writeJSONFile,
  INTERNET_FILE_PATH,
} from "../utils/fileUtils.js";
import { getUsers, getUserByUsername, saveUsers } from "../utils/userUtils.js";
import { logAction } from "../utils/logger.js";

export function setupGameHandlers(socket, io) {
  socket.on("scanIP", async ({ username, targetIP }) => {
    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const target = internet[targetIP];

    if (target) {
      logAction(username, `Scanned IP: ${targetIP}`);
      io.to(socket.id).emit("scanResult", { targetIP, details: target });
    } else {
      io.to(socket.id).emit("scanResult", {
        targetIP,
        error: "IP not found",
      });
    }
  });

  socket.on("hackIP", async ({ username, targetIP }) => {
    const users = await getUsers();
    const user = getUserByUsername(username);

    if (user) {
      const internet = await readJSONFile(INTERNET_FILE_PATH);
      const target = internet[targetIP];

      if (target) {
        if (
          user.tools.includes("exploit_kit") ||
          user.resources.cpu > target.securityLevel
        ) {
          // Successful hack
          user.resources = {
            ...user.resources,
            ...target.resources,
          };
          user.tools.push(...target.tools);

          await saveUsers(users);
          logAction(username, `Hacked IP: ${targetIP}`);
          io.to(socket.id).emit("hackResult", {
            success: true,
            targetIP,
            details: target,
          });
        } else {
          // Failed hack
          io.to(socket.id).emit("hackResult", {
            success: false,
            targetIP,
            error: "Hack failed",
          });
        }
      } else {
        io.to(socket.id).emit("hackResult", {
          targetIP,
          error: "IP not found",
        });
      }
    } else {
      io.to(socket.id).emit("hackResult", {
        targetIP,
        error: "You are a guest",
      });
    }
  });
}

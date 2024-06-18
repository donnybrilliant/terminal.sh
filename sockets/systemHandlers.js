import si from "systeminformation";
import { checkUser, checkTargetIP } from "../utils/userUtils.js";

// New event to fetch hardware info
export function setupSystemHandlers(socket) {
  socket.on("hardwareInfo", async ({ username, targetIP }) => {
    if (username && !targetIP) {
      const { user } = await checkUser(username);
      console.log(user);
      if (!user) {
        return socket.emit("hardwareResult", {
          success: false,
          message: "Hardware info failed",
          error: "User not found",
          data: null,
        });
      } else {
        const resources = Object.keys(user.resources).map((key) => {
          const total = user.resources[key];
          const used =
            user.usedResources && user.usedResources[key]
              ? user.usedResources[key]
              : 0;
          return `${key}: ${used}/${total}`;
        });

        return socket.emit("hardwareResult", {
          success: true,
          message: "Hardware info received",
          data: resources,
        });
      }
    }

    if (targetIP) {
      const targetServer = await checkTargetIP(targetIP);
      if (!targetServer) {
        return socket.emit("hardwareResult", {
          success: false,
          message: "Hardware info failed",
          error: "Target server not found",
        });
      }

      const resources = Object.keys(targetServer.resources).map((key) => {
        const total = targetServer.resources[key];
        const used =
          targetServer.usedResources && targetServer.usedResources[key]
            ? targetServer.usedResources[key]
            : 0;
        return `${key}: ${used}/${total}`;
      });

      return socket.emit("hardwareResult", {
        success: true,
        message: "Hardware info received",
        data: resources,
      });
    }

    /*       const cpuInfo = await si.cpu();
      const memInfo = await si.mem();
      const osInfo = await si.osInfo();

      // Sending hardware info back to client
      return socket.emit("hardwareResult", {
        success: true,
        message: "Hardware info received",
        data: { cpu: cpuInfo, memory: memInfo, os: osInfo },
      }); */
  });
}

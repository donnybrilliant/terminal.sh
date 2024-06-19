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

  socket.on("wallet", async ({ username }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("walletResult", {
        success: false,
        message: "User not found",
      });
    }
    socket.emit("walletResult", {
      success: true,
      data: user.wallet,
    });
  });

  socket.on("ifconfig", async ({ username }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("ifconfigResult", {
        success: false,
        message: "User not found",
      });
    }
    socket.emit("ifconfigResult", {
      success: true,
      data: {
        name: user.username,
        ip: user.ip,
        mac: user.mac,
      },
    });
  });

  socket.on("exploited", async ({ username }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("exploitedResult", {
        success: false,
        message: "User not found",
      });
    }
    socket.emit("exploitedResult", {
      success: true,
      data: user.exploitedServers,
    });
  });

  socket.on("tools", async ({ username }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("toolsResult", {
        success: false,
        message: "User not found",
      });
    }
    const tools = user.tools.map((tool) => ({
      name: tool.name,
      level: tool.level || "N/A",
      // more stuff here..
    }));
    socket.emit("toolsResult", {
      success: true,
      data: tools,
    });
  });

  socket.on("miners", async ({ username }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("minersResult", {
        success: false,
        message: "User not found",
      });
    }
    socket.emit("minersResult", {
      success: true,
      data: user.miners || [],
    });
  });

  socket.on("userinfo", async ({ username }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("userinfoResult", {
        success: false,
        message: "User not found",
      });
    }
    const userInfo = {
      name: user.username,
      level: user.level,
      achievements: user.achievements,
      inventory: user.inventory,
      experience: user.experience,
      ip: user.ip,
      wallet: user.wallet,
    };
    socket.emit("userinfoResult", {
      success: true,
      data: userInfo,
    });
  });
}

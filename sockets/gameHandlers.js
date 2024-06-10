export function setupGameHandlers(socket, io) {
  socket.on("scanIP", async ({ username, targetIP }) => {
    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const target = internet[targetIP];

    if (target) {
      logAction(username, `Scanned IP: ${targetIP}`);
      socket.emit("scanResult", {
        success: true,
        message: `Scan result for ${targetIP}`,
        error: null,
        data: target,
      });
    } else {
      socket.emit("scanResult", {
        success: false,
        message: "Scan failed",
        error: "IP not found",
        data: null,
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
          user.resources = {
            ...user.resources,
            ...target.resources,
          };
          user.tools.push(...target.tools);

          await saveUsers(users);
          logAction(username, `Hacked IP: ${targetIP}`);
          socket.emit("hackResult", {
            success: true,
            message: `Successfully hacked ${targetIP}`,
            error: null,
            data: target,
          });
        } else {
          socket.emit("hackResult", {
            success: false,
            message: "Hack failed",
            error: "Insufficient resources",
            data: null,
          });
        }
      } else {
        socket.emit("hackResult", {
          success: false,
          message: "Hack failed",
          error: "IP not found",
          data: null,
        });
      }
    }
  });

  socket.on("startMining", async ({ username, targetIP }) => {
    const users = await readJSONFile(USERS_FILE_PATH);
    const user = users.find((u) => u.username === username);
    if (!user) {
      return socket.emit("miningResult", {
        success: false,
        message: "Mining failed",
        error: "User not found",
        data: null,
      });
    }

    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const targetServer = internet[targetIP];
    if (!targetServer) {
      return socket.emit("miningResult", {
        success: false,
        message: "Mining failed",
        error: "Target server not found",
        data: null,
      });
    }

    if (
      targetServer.activeMiners >= Math.floor(targetServer.resources.cpu / 25)
    ) {
      return socket.emit("miningResult", {
        success: false,
        message: "Mining failed",
        error: "Not enough resources on target server",
        data: null,
      });
    }

    user.activeMiners.push({ targetIP, startTime: Date.now() });
    targetServer.activeMiners += 1;
    await Promise.all([
      writeJSONFile(USERS_FILE_PATH, users),
      writeJSONFile(INTERNET_FILE_PATH, internet),
    ]);

    socket.emit("miningResult", {
      success: true,
      message: "Mining started on target server",
      error: null,
      data: null,
    });
  });
}

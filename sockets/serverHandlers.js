// serverHandlers.js

import { createServer } from "../utils/serverFactory.js";
import {
  readJSONFile,
  writeJSONFile,
  INTERNET_FILE_PATH,
  USERS_FILE_PATH,
  SERVER_TEMPLATE,
} from "../utils/fileUtils.js";
import { generateLocalNetworkIP } from "../utils/ipUtils.js";

export function setupServerHandlers(socket) {
  socket.on("createServer", async ({ username }) => {
    try {
      const server = await createInternetServer(username);
      socket.emit("createServerResult", {
        success: true,
        message: "Server created successfully",
        data: server,
      });
    } catch (error) {
      socket.emit("createServerResult", {
        success: false,
        message: "Failed to create server",
        error: error.message,
      });
    }
  });

  socket.on("createLocalServer", async ({ username, targetIP }) => {
    try {
      const server = await createLocalServer(username, targetIP);
      socket.emit("createLocalServerResult", {
        success: true,
        message: "Local server created successfully",
        data: server,
      });
    } catch (error) {
      socket.emit("createLocalServerResult", {
        success: false,
        message: "Failed to create local server",
        error: error.message,
      });
    }
  });
}

export async function createInternetServer(username) {
  const internet = await readJSONFile(INTERNET_FILE_PATH);
  const users = await readJSONFile(USERS_FILE_PATH);
  const serverConfig = await readJSONFile(SERVER_TEMPLATE);

  const user = users.find((u) => u.username === username);
  if (!user) throw new Error("User not found.");

  const server = await createServer(serverConfig, internet, users);

  // Save server to the appropriate place in your JSON or database
  internet[server.ip] = server;
  await writeJSONFile(INTERNET_FILE_PATH, internet);

  return server;
}

export async function createLocalServer(username, targetIP) {
  const internet = await readJSONFile(INTERNET_FILE_PATH);
  const users = await readJSONFile(USERS_FILE_PATH);
  const serverConfig = await readJSONFile(SERVER_TEMPLATE);
  const usedIPs = new Set(
    Object.keys(internet).concat(
      ...Object.values(internet).map((server) =>
        Object.keys(server.localNetwork || {})
      )
    )
  );

  let localIP;
  let targetType;
  let baseIP;

  if (targetIP) {
    const server = internet[targetIP];
    if (!server) throw new Error("Server not found.");
    localIP = generateLocalNetworkIP(server.localIP, usedIPs);
    targetType = "server";
    baseIP = server.localIP;
  } else {
    const user = users.find((u) => u.username === username);
    if (!user) throw new Error("User not found.");
    localIP = generateLocalNetworkIP(user.localIP, usedIPs);
    targetType = "user";
    baseIP = user.localIP;
  }

  const serverConfigWithLocalIP = {
    ...serverConfig,
    ip: null,
    localIP,
  };

  const server = await createServer(
    serverConfigWithLocalIP,
    internet,
    users,
    baseIP,
    true
  );

  // Save server to the appropriate place in your JSON or database
  if (targetType === "user") {
    const user = users.find((u) => u.username === username);
    if (!user.localNetwork) {
      user.localNetwork = {};
    }
    user.localNetwork[localIP] = server;
    await writeJSONFile(USERS_FILE_PATH, users);
  } else {
    if (!internet[targetIP].localNetwork) {
      internet[targetIP].localNetwork = {};
    }
    internet[targetIP].localNetwork[localIP] = server;
    await writeJSONFile(INTERNET_FILE_PATH, internet);
  }

  return server;
}

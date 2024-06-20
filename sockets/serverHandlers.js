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
import { checkTargetIP } from "../utils/userUtils.js";

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

  socket.on("createLocalServer", async ({ username, targetIP, parentIP }) => {
    try {
      const server = await createLocalServer(username, targetIP, parentIP);
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

export async function createLocalServer(username, targetIP, parentIP) {
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

  console.log("Starting createLocalServer...");
  console.log("username:", username);
  console.log("targetIP:", targetIP);
  console.log("parentIP:", parentIP);

  let localIP;
  let targetType;
  let baseIP;
  let parentServer = null;

  if (targetIP) {
    const checkResult = await checkTargetIP(targetIP, parentIP);
    let { server: parentServer } = checkResult;
    console.log("Found parentServer:", parentServer);
    if (!parentServer) {
      throw new Error("Parent server not found for the specified target IP.");
    }
    if (!parentServer.localNetwork) {
      parentServer.localNetwork = {}; // Initialize localNetwork if it doesn't exist
    }
    localIP = generateLocalNetworkIP(parentServer.localIP, usedIPs);
    console.log("Generated localIP:", localIP);
    targetType = "server";
    baseIP = parentServer.localIP;
  } else {
    const user = users.find((u) => u.username === username);
    if (!user) throw new Error("User not found.");
    if (!user.localNetwork) {
      user.localNetwork = {}; // Initialize localNetwork if it doesn't exist
    }
    localIP = generateLocalNetworkIP(user.localIP, usedIPs);
    console.log("Generated localIP for user:", localIP);
    targetType = "user";
    baseIP = user.localIP;
  }

  const serverConfigWithLocalIP = {
    ...serverConfig,
    ip: null,
    localIP,
  };

  console.log("Creating server with config:", serverConfigWithLocalIP);

  const server = await createServer(
    serverConfigWithLocalIP,
    internet,
    users,
    baseIP,
    true
  );

  console.log("Created server:", server);

  if (targetType === "user") {
    const user = users.find((u) => u.username === username);
    user.localNetwork[localIP] = server;
    console.log("Updating user with new local server:", user);
    await writeJSONFile(USERS_FILE_PATH, users);
  } else if (parentServer) {
    parentServer.localNetwork[localIP] = server;
    console.log("Updating parent server with new local server:", parentServer);

    const updateNestedServer = (network, targetIP, updatedServer) => {
      if (network[targetIP]) {
        network[targetIP] = updatedServer;
      } else {
        for (const key in network) {
          if (network[key].localNetwork) {
            updateNestedServer(
              network[key].localNetwork,
              targetIP,
              updatedServer
            );
          }
        }
      }
    };

    updateNestedServer(internet, targetIP, parentServer);

    console.log("Writing to INTERNET_FILE_PATH:", INTERNET_FILE_PATH);
    await writeJSONFile(INTERNET_FILE_PATH, internet);
    console.log("Updated internet:", JSON.stringify(internet, null, 2));
  }

  return server;
}

// lanSniffing.js

import {
  writeJSONFile,
  readJSONFile,
  INTERNET_FILE_PATH,
} from "../utils/fileUtils.js";
import { createLocalServer } from "../sockets/serverHandlers.js";
import { logAction } from "../utils/logger.js";
import { checkTargetIP, saveUser } from "../utils/userUtils.js";

let lanSniffingIntervals = {};

export async function startLanSniffing(user, targetServer, targetIP, socket) {
  const snifferTool = user.tools.find((tool) => tool.name === "lan_sniffer");
  if (!snifferTool) {
    throw new Error("LAN Sniffer tool not found");
  }

  const snifferResourceUsage = snifferTool.resources;

  if (!targetServer.usedResources) {
    targetServer.usedResources = { cpu: 0, bandwidth: 0, ram: 0 };
  }

  if (!targetServer.activeSniffers) {
    targetServer.activeSniffers = {};
  }

  const availableResources = {
    cpu: targetServer.resources.cpu - targetServer.usedResources.cpu,
    bandwidth:
      targetServer.resources.bandwidth - targetServer.usedResources.bandwidth,
    ram: targetServer.resources.ram - targetServer.usedResources.ram,
  };

  if (
    availableResources.cpu < snifferResourceUsage.cpu ||
    availableResources.bandwidth < snifferResourceUsage.bandwidth ||
    availableResources.ram < snifferResourceUsage.ram
  ) {
    throw new Error("Not enough resources on target server");
  }

  if (!user.activeSniffers) {
    user.activeSniffers = {};
  }

  if (!user.activeSniffers[targetIP]) {
    user.activeSniffers[targetIP] = {
      startTime: Date.now(),
      resourceUsage: snifferResourceUsage,
    };
  }

  targetServer.activeSniffers[user.ip] = {
    startTime: Date.now(),
    resourceUsage: snifferResourceUsage,
  };
  targetServer.usedResources.cpu += snifferResourceUsage.cpu;
  targetServer.usedResources.bandwidth += snifferResourceUsage.bandwidth;
  targetServer.usedResources.ram += snifferResourceUsage.ram;

  await Promise.all([
    saveUser(user),
    updateInternetFile(targetServer, targetIP),
  ]);

  logAction(user.username, `Started LAN sniffing on: ${targetIP}`);
  startLanSniffingTimer(user, targetIP, socket);
  socket.emit("lanSnifferResult", {
    success: true,
    message: "LAN sniffing started on target server",
    error: null,
    data: null,
  });
}

function startLanSniffingTimer(user, targetIP, socket) {
  const snifferInterval = setTimeout(async () => {
    const server = await createLocalServer(user.username, targetIP);
    socket.emit("lanSnifferResult", {
      success: true,
      message: `LAN sniffing detected server: ${server.localIP}`,
      data: server,
    });

    // Remove resources and stop LAN sniffing
    const { server: targetServer } = await checkTargetIP(targetIP);
    await stopLanSniffing(user, targetServer, targetIP, socket);
  }, 60000); // Timeout set to 1 minute

  lanSniffingIntervals[user.id] = snifferInterval;
}

export async function stopLanSniffing(user, targetServer, targetIP, socket) {
  if (
    lanSniffingIntervals[user.id] &&
    user.activeSniffers &&
    user.activeSniffers[targetIP]
  ) {
    clearTimeout(lanSniffingIntervals[user.id]);
    delete lanSniffingIntervals[user.id];

    // Restore resources when LAN sniffing stops
    const snifferResourceUsage = user.activeSniffers[targetIP].resourceUsage;
    targetServer.usedResources.cpu = Math.max(
      targetServer.usedResources.cpu - snifferResourceUsage.cpu,
      0
    );
    targetServer.usedResources.bandwidth = Math.max(
      targetServer.usedResources.bandwidth - snifferResourceUsage.bandwidth,
      0
    );
    targetServer.usedResources.ram = Math.max(
      targetServer.usedResources.ram - snifferResourceUsage.ram,
      0
    );
    delete targetServer.activeSniffers[user.ip];

    delete user.activeSniffers[targetIP];

    await Promise.all([
      saveUser(user),
      updateInternetFile(targetServer, targetIP),
    ]);

    logAction(user.username, `Stopped LAN sniffing on: ${targetIP}`);
    socket.emit("lanSnifferResult", {
      success: true,
      message: `LAN sniffing stopped on ${targetIP}.`,
    });
  } else {
    socket.emit("lanSnifferResult", {
      success: false,
      message: `No active LAN sniffing found on ${targetIP}.`,
    });
  }
}

async function updateInternetFile(targetServer, targetIP) {
  const internetData = await readJSONFile(INTERNET_FILE_PATH);
  internetData[targetIP] = targetServer;
  await writeJSONFile(INTERNET_FILE_PATH, internetData);
}

import { writeJSONFile, INTERNET_FILE_PATH } from "../utils/fileUtils.js";
import { saveUser } from "../utils/userUtils.js";

let miningIntervals = {};

export function startMiningTimer(user, targetIP, socket) {
  const miningInterval = setInterval(async () => {
    const now = Date.now();
    const elapsedTime = (now - user.activeMiners[targetIP].startTime) / 1000; // seconds

    const cryptoMined = elapsedTime * 0.1; // Assuming 0.1 crypto per second
    user.resources.crypto += cryptoMined;
    user.activeMiners[targetIP].startTime = now; // reset the start time

    await saveUser(user);

    socket.emit("miningUpdate", {
      success: true,
      message: `Mining ongoing on ${targetIP}. Earned ${cryptoMined} crypto.`,
    });
  }, 5000); // Update every 5 seconds

  miningIntervals[user.id] = miningInterval;
}

export async function startMining(user, targetServer, targetIP, socket) {
  const minerTool = user.tools.find((tool) => tool.name === "Crypto Miner");
  if (!minerTool) {
    throw new Error("Crypto Miner tool not found");
  }

  const minerResourceUsage = minerTool.resources;

  if (!targetServer.usedResources) {
    targetServer.usedResources = { cpu: 0, bandwidth: 0, ram: 0 };
  }

  if (!targetServer.activeMiners) {
    targetServer.activeMiners = {};
  }

  const availableResources = {
    cpu: targetServer.resources.cpu - targetServer.usedResources.cpu,
    bandwidth:
      targetServer.resources.bandwidth - targetServer.usedResources.bandwidth,
    ram: targetServer.resources.ram - targetServer.usedResources.ram,
  };

  if (
    availableResources.cpu < minerResourceUsage.cpu ||
    availableResources.bandwidth < minerResourceUsage.bandwidth ||
    availableResources.ram < minerResourceUsage.ram
  ) {
    throw new Error("Not enough resources on target server");
  }

  if (!user.activeMiners) {
    user.activeMiners = {};
  }

  if (!user.activeMiners[targetIP]) {
    user.activeMiners[targetIP] = {
      startTime: Date.now(),
      resourceUsage: minerResourceUsage,
    };
  }

  targetServer.activeMiners[user.ip] = {
    startTime: Date.now(),
    resourceUsage: minerResourceUsage,
  };
  targetServer.usedResources.cpu += minerResourceUsage.cpu;
  targetServer.usedResources.bandwidth += minerResourceUsage.bandwidth;
  targetServer.usedResources.ram += minerResourceUsage.ram;

  await Promise.all([
    saveUser(user),
    writeJSONFile(INTERNET_FILE_PATH, {
      ...targetServer,
      [targetIP]: targetServer,
    }),
  ]);

  startMiningTimer(user, targetIP, socket);

  socket.emit("miningResult", {
    success: true,
    message: "Mining started on target server",
    error: null,
    data: null,
  });
}

export async function stopMining(user, targetServer, targetIP, socket) {
  if (
    miningIntervals[user.id] &&
    user.activeMiners &&
    user.activeMiners[targetIP]
  ) {
    clearInterval(miningIntervals[user.id]);
    delete miningIntervals[user.id];

    // Restore resources when mining stops
    const minerResourceUsage = user.activeMiners[targetIP].resourceUsage;
    targetServer.usedResources.cpu = Math.max(
      targetServer.usedResources.cpu - minerResourceUsage.cpu,
      0
    );
    targetServer.usedResources.bandwidth = Math.max(
      targetServer.usedResources.bandwidth - minerResourceUsage.bandwidth,
      0
    );
    targetServer.usedResources.ram = Math.max(
      targetServer.usedResources.ram - minerResourceUsage.ram,
      0
    );
    delete targetServer.activeMiners[user.ip];

    delete user.activeMiners[targetIP];

    await Promise.all([
      saveUser(user),
      writeJSONFile(INTERNET_FILE_PATH, {
        ...targetServer,
        [targetIP]: targetServer,
      }),
    ]);

    socket.emit("miningUpdate", {
      success: true,
      message: `Mining stopped on ${targetIP}.`,
    });
  } else {
    socket.emit("miningUpdate", {
      success: false,
      message: `No active mining found on ${targetIP}.`,
    });
  }
}

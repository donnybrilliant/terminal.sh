import si from "systeminformation";

// New event to fetch hardware info
export function setupSystemHandlers(socket) {
  socket.on("requestHardwareInfo", async () => {
    try {
      const cpuInfo = await si.cpu();
      const memInfo = await si.mem();
      const osInfo = await si.osInfo();

      // Sending hardware info back to client
      socket.emit("hardwareInfo", {
        cpu: cpuInfo,
        memory: memInfo,
        os: osInfo,
      });
    } catch (error) {
      console.error("Failed to get hardware info:", error);
    }
  });
}

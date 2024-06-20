import {
  readJSONFile,
  writeJSONFile,
  INTERNET_FILE_PATH,
  TOOLS_FILE_PATH,
} from "../utils/fileUtils.js";
import { checkUser, checkTargetIP, saveUser } from "../utils/userUtils.js";
import { logAction } from "../utils/logger.js";
import { startMining, stopMining } from "../game/mining.js";
import { startLanSniffing, stopLanSniffing } from "../game/sniffing.js";
import {
  getToolData,
  getFileFromPath,
  mergeTools,
  generateUniqueFileName,
} from "../utils/toolUtils.js";

export function setupGameHandlers(socket, io) {
  socket.on("scanInternet", async ({ username }) => {
    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const ipAddresses = Object.keys(internet);

    logAction(username, "Scanned internet for IP addresses");
    socket.emit("scanInternetResult", {
      success: true,
      message: "Scan result for internet",
      error: null,
      data: ipAddresses,
    });
  });

  socket.on("scanIP", async ({ username, targetIP, parentIP }) => {
    const { server: targetServer } = await checkTargetIP(targetIP, parentIP);

    if (targetServer) {
      logAction(username, `Scanned IP: ${targetIP}`);
      socket.emit("scanIPResult", {
        success: true,
        message: `Scan result for ${targetIP}`,
        error: null,
        data: targetServer,
      });
    } else {
      return socket.emit("scanIPResult", {
        success: false,
        message: "Scan failed",
        error: "IP not found",
        data: null,
      });
    }
  });

  socket.on("scanConnectedIPs", async ({ username, targetIP, parentIP }) => {
    const { server: targetServer } = await checkTargetIP(targetIP, parentIP);

    if (targetServer) {
      logAction(username, `Scanned connected IPs on: ${targetIP}`);
      socket.emit("scanIPResult", {
        success: true,
        message: `Connected IPs for ${targetIP}`,
        error: null,
        data: targetServer.connectedIPs,
      });
    } else {
      return socket.emit("scanIPResult", {
        success: false,
        message: "Scan failed",
        error: "IP not found",
        data: null,
      });
    }
  });

  socket.on("startMining", async ({ username, targetIP }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("miningResult", {
        success: false,
        message: "Mining failed",
        error: "User not found",
        data: null,
      });
    }
    const { server: targetServer } = await checkTargetIP(targetIP);
    if (!targetServer) {
      return socket.emit("miningResult", {
        success: false,
        message: "Mining failed",
        error: "Target server not found",
        data: null,
      });
    }

    await startMining(user, targetServer, targetIP, socket);
  });

  socket.on("stopMining", async ({ username, targetIP }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("miningResult", {
        success: false,
        message: "Stopping mining failed",
        error: "User not found",
        data: null,
      });
    }
    const { server: targetServer } = await checkTargetIP(targetIP);
    if (!targetServer) {
      return socket.emit("miningResult", {
        success: false,
        message: "Mining failed",
        error: "Target server not found",
      });
    }

    await stopMining(user, targetServer, targetIP, socket);
  });

  socket.on("getTool", async ({ username, targetIP, parentIP, toolName }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("getToolResult", {
        success: false,
        message: "Download failed",
        error: "User not found",
      });
    }

    const { server: targetServer } = await checkTargetIP(targetIP, parentIP);
    if (!targetServer) {
      return socket.emit("getToolResult", {
        success: false,
        message: "Download failed",
        error: "Target server not found",
      });
    }

    if (!targetServer.tools.includes(toolName)) {
      return socket.emit("getToolResult", {
        success: false,
        message: "Download failed",
        error: "Tool not found on target server",
      });
    }

    const newTool = await getToolData(toolName);
    if (!newTool) {
      return socket.emit("getToolResult", {
        success: false,
        message: "Download failed",
        error: "Tool data not found",
      });
    }

    const currentTool = user.tools.find((tool) => tool.name === newTool.name);

    if (!currentTool) {
      // Check if the new tool is complete
      if (newTool.isPatch) {
        return socket.emit("getToolResult", {
          success: false,
          message: "Download failed",
          error: "Patch not installed. Base tool is required first.",
        });
      }

      user.tools.push(newTool);
      user.home.bin = user.home.bin || {};
      user.home.bin[newTool.name] = newTool;
      await saveUser(user);
      return socket.emit("getToolResult", {
        success: true,
        message: `${newTool.name} downloaded successfully from ${targetIP}`,
        tool: newTool,
      });
    } else {
      const mergedTool = mergeTools(currentTool, newTool);
      user.tools = user.tools.filter((tool) => tool.name !== newTool.name);
      user.tools.push(mergedTool);
      user.home.bin[mergedTool.name] = mergedTool;
      await saveUser(user);
      return socket.emit("getToolResult", {
        success: true,
        message: `${mergedTool.name} upgraded successfully from ${targetIP}`,
        tool: mergedTool,
      });
    }
  });

  socket.on("download", async ({ username, targetIP, parentIP, filePath }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("downloadResult", {
        success: false,
        message: "Download failed",
        error: "User not found",
      });
    }
    const { server: targetServer } = await checkTargetIP(targetIP, parentIP);
    if (!targetServer) {
      return socket.emit("downloadResult", {
        success: false,
        message: "Download failed",
        error: "Target server not found",
      });
    }

    const fileData = getFileFromPath(targetServer.fileSystem, filePath);

    if (!fileData) {
      return socket.emit("downloadResult", {
        success: false,
        message: "Download failed",
        error: "File not found",
      });
    } else if (!fileData.isDownloadable) {
      return socket.emit("downloadResult", {
        success: false,
        message: "Download failed",
        error: "File is not downloadable",
      });
    }

    if (fileData.isTool) {
      const currentTool = user.tools.find(
        (tool) => tool.name === fileData.name
      );

      if (!currentTool) {
        // Check if the new tool is complete
        if (fileData.isPatch) {
          return socket.emit("getToolResult", {
            success: false,
            message: "Download failed",
            error: "Patch not installed. Base tool is required first.",
          });
        }

        user.tools.push(fileData);
        user.home.bin = user.home.bin || {};
        user.home.bin[fileData.name] = fileData;
        await saveUser(user);
        return socket.emit("getToolResult", {
          success: true,
          message: `${fileData.name} downloaded successfully from ${targetIP}`,
          tool: fileData,
        });
      } else {
        const mergedTool = mergeTools(currentTool, fileData);
        user.tools = user.tools.filter((tool) => tool.name !== fileData.name);
        user.tools.push(mergedTool);
        user.home.bin[fileData.name] = mergedTool;
        await saveUser(user);
        return socket.emit("getToolResult", {
          success: true,
          message: `${fileData.name} upgraded successfully from ${targetIP}`,
          tool: mergedTool,
        });
      }
    } else {
      let fileName = filePath.split("/").pop();
      fileName = generateUniqueFileName(user.home.downloads, fileName);
      user.home.downloads = user.home.downloads || {};
      user.home.downloads[fileName] = fileData;

      await saveUser(user);
      return socket.emit("downloadResult", {
        success: true,
        message: `${filePath} downloaded successfully from ${targetIP}`,
        file: fileData,
      });
    }
  });

  socket.on("ssh_exploit", async ({ username, targetIP, parentIP }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("sshExploitResult", {
        success: false,
        message: "Exploit failed",
        error: "User not found",
        data: null,
      });
    }

    const checkResult = await checkTargetIP(targetIP, parentIP);
    const { server: targetServer, path: exploitationPath } = checkResult;
    if (!targetServer) {
      return socket.emit("sshExploitResult", {
        success: false,
        message: "Exploit failed",
        error: "Target IP not found",
        data: null,
      });
    }

    const sshService = targetServer.services.find(
      (service) => service.name === "ssh"
    );
    if (!sshService) {
      return socket.emit("sshExploitResult", {
        success: false,
        message: "Exploit failed",
        error: "SSH service not found",
        data: null,
      });
    }

    const vulnerabilities = sshService.vulnerabilities;
    const matchingExploits = user.tools
      .flatMap((tool) => tool.exploits || [])
      .filter((exploit) =>
        vulnerabilities.some(
          (vul) => vul.type === exploit.type && exploit.level >= vul.level
        )
      );

    if (matchingExploits.length === 0) {
      return socket.emit("sshExploitResult", {
        success: false,
        message: "Exploit failed",
        error: "No matching vulnerabilities found",
        data: null,
      });
    }

    user.exploitedServers = user.exploitedServers || {};
    user.exploitedServers[exploitationPath] =
      user.exploitedServers[exploitationPath] || {};
    user.exploitedServers[exploitationPath][sshService.name] =
      user.exploitedServers[exploitationPath][sshService.name] || [];

    matchingExploits.forEach((exploit) => {
      if (
        !user.exploitedServers[exploitationPath][sshService.name].includes(
          exploit.type
        )
      ) {
        user.exploitedServers[exploitationPath][sshService.name].push(
          exploit.type
        );
      }
    });

    await saveUser(user);

    logAction(username, `Exploited SSH on IP: ${targetIP}`);
    socket.emit("sshExploitResult", {
      success: true,
      message: `Successfully exploited SSH on ${targetIP}`,
      error: null,
      data: targetServer,
    });
  });

  socket.on("password_sniffer", async ({ username, targetIP, parentIP }) => {
    try {
      const { user } = await checkUser(username);

      if (!user) {
        return socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "User not found",
          data: null,
        });
      }

      const { server: targetServer } = await checkTargetIP(targetIP, parentIP);

      if (!targetServer) {
        return socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "Target IP not found",
          data: null,
        });
      }

      const sshService = targetServer.services.find(
        (service) => service.name === "ssh"
      );

      if (!sshService) {
        return socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "SSH service not found",
          data: null,
        });
      }

      // handle this differently
      const passwordVulnerability = sshService.vulnerabilities.find(
        (vul) => vul.type === "password"
      );

      if (!passwordVulnerability) {
        return socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "Password vulnerability not found",
          data: null,
        });
      }

      const requiredVulnerabilities = sshService.vulnerabilities
        .filter((vul) => vul.type !== "password")
        .map((vul) => vul.type);

      // Ensure exploitedServers is initialized
      user.exploitedServers = user.exploitedServers || {};
      user.exploitedServers[targetIP] = user.exploitedServers[targetIP] || {};
      user.exploitedServers[targetIP].ssh =
        user.exploitedServers[targetIP].ssh || [];
      user.exploitedServers[targetIP].roles =
        user.exploitedServers[targetIP].roles || [];

      const exploitedVulnerabilities = user.exploitedServers[targetIP].ssh;

      const allRequiredExploited = requiredVulnerabilities.every((vul) =>
        exploitedVulnerabilities.includes(vul)
      );

      if (!allRequiredExploited) {
        return socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "Prerequisite vulnerabilities not exploited",
          data: null,
        });
      }

      const tool = user.tools.find((tool) => tool.name === "Password Sniffer");
      const toolLevel = tool ? tool.level : 0;
      const availableRoles = targetServer.roles
        .filter((role) => role.level <= toolLevel)
        .sort((a, b) => a.level - b.level);

      if (availableRoles.length === 0) {
        return socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "No user roles can be accessed with the current tool level",
          data: null,
        });
      }

      const accessedRole = availableRoles[0];

      // Add the password vulnerability to the list if not already there
      if (!user.exploitedServers[targetIP].ssh.includes("password")) {
        user.exploitedServers[targetIP].ssh.push("password");
      }

      // This should maybe be for a rootkit type thing?
      // Add the accessed role to the list of roles if not already there
      if (
        !user.exploitedServers[targetIP].roles.some(
          (role) => role.role === accessedRole.role
        )
      ) {
        user.exploitedServers[targetIP].roles.push({
          role: accessedRole.role,
          level: accessedRole.level,
        });
      }

      await saveUser(user);

      const remainingRoles = targetServer.roles.filter(
        (role) => role.level > toolLevel
      ).length;

      const responseMessage =
        remainingRoles > 0
          ? `Password cracked for role ${accessedRole.role}. There are more roles to be cracked.`
          : `Password cracked for role ${accessedRole.role}. All roles have been cracked.`;

      logAction(username, `Password cracked on IP: ${targetIP}`);
      socket.emit("passwordSnifferResult", {
        success: true,
        message: responseMessage,
        error: null,
        data: targetServer,
      });
    } catch (error) {
      console.error("Error in password_sniffer handler:", error);
      socket.emit("passwordSnifferResult", {
        success: false,
        message: "Password sniffer failed",
        error: "Internal server error",
        data: null,
      });
    }
  });

  socket.on("ssh", async ({ username, targetIP, parentIP }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("sshResult", {
        success: false,
        message: "SSH Connection failed",
        error: "User not found",
      });
    }

    const checkResult = await checkTargetIP(targetIP, parentIP);
    const { server: targetServer, path: exploitationPath } = checkResult;
    console.log(exploitationPath);

    if (!targetServer) {
      return socket.emit("sshResult", {
        success: false,
        message: "SSH connection failed",
        error: "Target IP not found",
      });
    }

    // Check for SSH exploits
    const sshService = targetServer.services.find(
      (service) => service.name === "ssh"
    );
    if (!sshService) {
      return socket.emit("sshResult", {
        success: false,
        message: "SSH Connection failed",
        error: "Service not found",
      });
    }

    if (!sshService.vulnerable) {
      return socket.emit("sshResult", {
        success: false,
        message: "SSH connection failed",
        error: "SSH service not vulnerable",
      });
    }

    // Check required vulnerabilities
    user.exploitedServers = user.exploitedServers || {};
    user.exploitedServers[exploitationPath] =
      user.exploitedServers[exploitationPath] || {};
    const exploitedVulnerabilities =
      user.exploitedServers[exploitationPath].ssh || [];
    const exploitedRoles = user.exploitedServers[exploitationPath].roles || [];

    const requiredVulnerabilities = sshService.vulnerabilities.map(
      (vul) => vul.type
    );
    const allRequiredExploited = requiredVulnerabilities.every((vul) =>
      exploitedVulnerabilities.includes(vul)
    );

    if (!allRequiredExploited) {
      return socket.emit("sshResult", {
        success: false,
        message: "SSH Connection failed",
        error: "Prerequisite vulnerabilities not exploited",
      });
    }

    // Should check user role and permissions here in the future
    // Check if any role is exploited
    const hasExploitedRole = exploitedRoles.length > 0;
    let eventData = null;
    if (hasExploitedRole) {
      eventData = targetServer;
    }

    logAction(username, `Connected to SSH on IP: ${targetIP}`);
    socket.emit("sshResult", {
      success: true,
      message: `Connected to ${targetIP}...`,
      data: eventData,
      targetIP,
      ssh: true,
    });

    await saveUser(user);
  });

  socket.on("user_enum", async ({ username, targetIP, parentIP }) => {
    const { user } = await checkUser(username);

    if (!user) {
      socket.emit("userEnumResult", {
        success: false,
        message: "Enumeration failed",
        error: "User not found",
      });
      return;
    }

    const { server: targetServer } = await checkTargetIP(targetIP, parentIP);

    if (!targetServer) {
      socket.emit("userEnumResult", {
        success: false,
        message: "User Enumeration failed",
        error: "Target IP not found",
      });
      return;
    }

    logAction(username, `Enumerated users on IP: ${targetIP}`);
    socket.emit("userEnumResult", {
      success: true,
      message: `User enumeration on ${targetIP} succeeded`,
      data: targetServer.roles,
    });
  });

  socket.on(
    "password_cracker",
    async ({ username, targetIP, parentIP, role }) => {
      const { user } = await checkUser(username);

      if (!user) {
        return socket.emit("passwordCrackerResult", {
          success: false,
          message: "Cracking failed",
          error: "User not found",
        });
      }
      const checkResult = await checkTargetIP(targetIP, parentIP);
      const { server: targetServer, path: exploitationPath } = checkResult;
      if (!targetServer) {
        return socket.emit("passwordCrackerResult", {
          success: false,
          message: "Cracking failed",
          error: "Target IP not found",
        });
      }

      const sshService = targetServer.services.find(
        (service) => service.name === "ssh"
      );

      if (!sshService) {
        return socket.emit("passwordCrackerResult", {
          success: false,
          message: "Cracking failed",
          error: "SSH service not found",
        });
      }

      const requiredVulnerabilities = sshService.vulnerabilities.map(
        (vul) => vul.type
      );
      user.exploitedServers = user.exploitedServers || {};
      const exploitedVulnerabilities =
        user.exploitedServers[exploitationPath]?.ssh || [];

      const allRequiredExploited = requiredVulnerabilities.every((vul) =>
        exploitedVulnerabilities.includes(vul)
      );

      if (!allRequiredExploited) {
        return socket.emit("passwordCrackerResult", {
          success: false,
          message: "Cracking failed",
          error: "Prerequisite vulnerabilities not exploited",
        });
      }

      const roleDetails = targetServer.roles.find((r) => r.role === role);

      if (!roleDetails) {
        return socket.emit("passwordCrackerResult", {
          success: false,
          message: "Cracking failed",
          error: "Role not found",
        });
      }

      // Change to exploits.type === password?
      const tool = user.tools.find((tool) => tool.name === "password_cracker");
      if (!tool || roleDetails.level > tool.exploits[0].level) {
        return socket.emit("passwordCrackerResult", {
          success: false,
          message: "Cracking failed",
          error: "Role level too high for the tool",
        });
      }

      logAction(
        username,
        `Cracked password for role ${role} on IP: ${targetIP}`
      );
      socket.emit("passwordCrackerResult", {
        success: true,
        message: `Password cracked for role ${role} on ${targetIP}`,
        data: targetServer,
        load: true,
      });
    }
  );

  socket.on("rootkit", async ({ username, targetIP, parentIP, role }) => {
    const { user } = await checkUser(username);

    if (!user) {
      socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "User not found",
      });
      return;
    }

    const { server: targetServer } = await checkTargetIP(targetIP, parentIP);
    if (!targetServer) {
      return socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "Target IP not found",
      });
    }

    const sshService = targetServer.services.find(
      (service) => service.name === "ssh"
    );

    if (!sshService) {
      return socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "SSH service not found",
      });
    }

    user.exploitedServers = user.exploitedServers || {};
    const exploitedKey = parentIP
      ? `${parentIP}.localNetwork.${targetIP}`
      : targetIP;
    user.exploitedServers[exploitedKey] =
      user.exploitedServers[exploitedKey] || {};
    user.exploitedServers[exploitedKey].roles =
      user.exploitedServers[exploitedKey].roles || [];

    // Check if the role is already present
    if (
      user.exploitedServers[exploitedKey].roles.some((r) => r.role === role)
    ) {
      return socket.emit("rootkitResult", {
        success: true,
        message: "Rootkit already installed",
      });
    }

    const requiredVulnerabilities = sshService.vulnerabilities.map(
      (vul) => vul.type
    );

    const exploitedVulnerabilities =
      user.exploitedServers[exploitedKey].ssh || [];

    const allRequiredExploited = requiredVulnerabilities.every((vul) =>
      exploitedVulnerabilities.includes(vul)
    );

    if (!allRequiredExploited) {
      return socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "Prerequisite vulnerabilities not exploited",
      });
    }

    const roleDetails = targetServer.roles.find((r) => r.role === role);

    if (!roleDetails) {
      return socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "Role not found",
      });
    }

    // Add the role if not already present
    user.exploitedServers[exploitedKey].roles.push({
      role,
      level: roleDetails.level,
    });

    await saveUser(user);

    logAction(
      username,
      `Rootkit initialized for role ${role} on IP: ${targetIP}`
    );
    socket.emit("rootkitResult", {
      success: true,
      message: `Rootkit initialized for role ${role} on ${targetIP}`,
    });
  });

  socket.on("startLanSniffing", async ({ username, targetIP }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("lanSnifferResult", {
        success: false,
        message: "LAN sniffing failed",
        error: "User not found",
        data: null,
      });
    }
    const { server: targetServer } = await checkTargetIP(targetIP);
    if (!targetServer) {
      return socket.emit("lanSnifferResult", {
        success: false,
        message: "LAN sniffing failed",
        error: "Target server not found",
        data: null,
      });
    }

    await startLanSniffing(user, targetServer, targetIP, socket);
  });

  socket.on("stopLanSniffing", async ({ username, targetIP }) => {
    const { user } = await checkUser(username);
    if (!user) {
      return socket.emit("lanSnifferResult", {
        success: false,
        message: "Stopping LAN sniffing failed",
        error: "User not found",
        data: null,
      });
    }
    const { server: targetServer } = await checkTargetIP(targetIP);
    if (!targetServer) {
      return socket.emit("lanSnifferResult", {
        success: false,
        message: "LAN sniffing failed",
        error: "Target server not found",
      });
    }

    await stopLanSniffing(user, targetServer, targetIP, socket);
  });
}

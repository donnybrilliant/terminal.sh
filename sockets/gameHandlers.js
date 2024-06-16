import {
  readJSONFile,
  writeJSONFile,
  INTERNET_FILE_PATH,
  USERS_FILE_PATH,
  TOOLS_FILE_PATH,
} from "../utils/fileUtils.js";
import { getUsers, getUserByUsername, saveUsers } from "../utils/userUtils.js";
import { logAction } from "../utils/logger.js";

function getFileFromPath(fileSystem, filePath) {
  const pathParts = filePath.split("/");
  let currentDir = fileSystem;

  for (const part of pathParts) {
    if (currentDir[part]) {
      currentDir = currentDir[part];
    } else {
      return null;
    }
  }

  return currentDir;
}

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

  socket.on("scanIP", async ({ username, targetIP }) => {
    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const target = internet[targetIP];

    if (target) {
      logAction(username, `Scanned IP: ${targetIP}`);
      socket.emit("scanIPResult", {
        success: true,
        message: `Scan result for ${targetIP}`,
        error: null,
        data: target,
      });
    } else {
      socket.emit("scanIPResult", {
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
    console.log(Math.floor(targetServer.resources.cpu / 25));
    console.log(targetServer.activeMiners);

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

  socket.on("download", async ({ username, targetIP, toolName, filePath }) => {
    const users = await readJSONFile(USERS_FILE_PATH);
    const user = users.find((u) => u.username === username);
    if (!user) {
      return socket.emit("downloadResult", {
        success: false,
        message: "Download failed",
        error: "User not found",
      });
    }

    if (targetIP && toolName) {
      // Handle tool download
      const internet = await readJSONFile(INTERNET_FILE_PATH);
      const targetServer = internet[targetIP];
      if (!targetServer) {
        return socket.emit("downloadResult", {
          success: false,
          message: "Download failed",
          error: "Target server not found",
        });
      }

      const tools = await readJSONFile(TOOLS_FILE_PATH);
      if (!targetServer.tools.includes(toolName)) {
        return socket.emit("downloadResult", {
          success: false,
          message: "Download failed",
          error: "Tool not found on target server",
        });
      }

      const newTool = tools.tools[toolName];
      if (!newTool) {
        return socket.emit("downloadResult", {
          success: false,
          message: "Download failed",
          error: "Tool data not found",
        });
      }

      const currentTool = user.tools.find((tool) => tool.name === toolName);
      if (!currentTool || (currentTool && currentTool.level < newTool.level)) {
        user.tools = user.tools.filter((tool) => tool.name !== toolName); // Remove existing tool if present
        user.tools.push(newTool);
        user.home.bin = user.home.bin || {};
        user.home.bin[toolName] = newTool;
      }

      await writeJSONFile(USERS_FILE_PATH, users);
      return socket.emit("downloadResult", {
        success: true,
        message: `${newTool.name} downloaded successfully`,
        toolName,
      });
    }

    if (filePath) {
      // Handle file download
      // Check if it is a tool - then add it to the user.tools
      const targetServer = await getFileFromPath(fileData, filePath);
      if (!targetServer || !targetServer.isDownloadable) {
        return socket.emit("downloadResult", {
          success: false,
          message: "Download failed",
          error: "File not found or not downloadable",
        });
      }

      user.home.downloads = user.home.downloads || {};
      user.home.downloads[filePath] = targetServer.content;

      await writeJSONFile(USERS_FILE_PATH, users);
      return socket.emit("downloadResult", {
        success: true,
        message: `${filePath} downloaded successfully`,
        fileName: filePath,
      });
    }

    return socket.emit("downloadResult", {
      success: false,
      message: "Download failed",
      error: "No valid download target specified",
    });
  });

  socket.on("ssh_exploit", async ({ username, targetIP }) => {
    const users = await getUsers();
    const user = getUserByUsername(username);

    if (!user) {
      socket.emit("sshExploitResult", {
        success: false,
        message: "Exploit failed",
        error: "User not found",
        data: null,
      });
      return;
    }

    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const target = internet[targetIP];

    if (!target) {
      socket.emit("sshExploitResult", {
        success: false,
        message: "Exploit failed",
        error: "Target IP not found",
        data: null,
      });
      return;
    }

    const sshService = target.services.find(
      (service) => service.name === "ssh"
    );

    if (!sshService) {
      socket.emit("sshExploitResult", {
        success: false,
        message: "Exploit failed",
        error: "SSH service not found",
        data: null,
      });
      return;
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
      socket.emit("sshExploitResult", {
        success: false,
        message: "Exploit failed",
        error: "No matching vulnerabilities found",
        data: null,
      });
      return;
    }

    // Add detailed information to exploitedServers
    // add this on ssh connection instead?
    user.exploitedServers = user.exploitedServers || {};
    user.exploitedServers[targetIP] = user.exploitedServers[targetIP] || {};
    user.exploitedServers[targetIP][sshService.name] =
      user.exploitedServers[targetIP][sshService.name] || [];

    matchingExploits.forEach((exploit) => {
      if (
        !user.exploitedServers[targetIP][sshService.name].includes(exploit.type)
      ) {
        user.exploitedServers[targetIP][sshService.name].push(exploit.type);
      }
    });

    await writeJSONFile(USERS_FILE_PATH, users);

    logAction(username, `Exploited SSH on IP: ${targetIP}`);
    socket.emit("sshExploitResult", {
      success: true,
      message: `Successfully exploited SSH on ${targetIP}`,
      error: null,
      data: target,
    });
  });

  socket.on("password_sniffer", async ({ username, targetIP }) => {
    try {
      const users = await getUsers();
      const user = getUserByUsername(username);

      if (!user) {
        socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "User not found",
          data: null,
        });
        return;
      }

      const internet = await readJSONFile(INTERNET_FILE_PATH);
      const target = internet[targetIP];

      if (!target) {
        socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "Target IP not found",
          data: null,
        });
        return;
      }

      const sshService = target.services.find(
        (service) => service.name === "ssh"
      );

      if (!sshService) {
        socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "SSH service not found",
          data: null,
        });
        return;
      }

      const passwordVulnerability = sshService.vulnerabilities.find(
        (vul) => vul.type === "password"
      );

      if (!passwordVulnerability) {
        socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "Password vulnerability not found",
          data: null,
        });
        return;
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
        socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "Prerequisite vulnerabilities not exploited",
          data: null,
        });
        return;
      }

      const tool = user.tools.find((tool) => tool.name === "Password Sniffer");
      const toolLevel = tool ? tool.level : 0;
      const availableRoles = target.roles
        .filter((role) => role.level <= toolLevel)
        .sort((a, b) => a.level - b.level);

      if (availableRoles.length === 0) {
        socket.emit("passwordSnifferResult", {
          success: false,
          message: "Password sniffer failed",
          error: "No user roles can be accessed with the current tool level",
          data: null,
        });
        return;
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

      await writeJSONFile(USERS_FILE_PATH, users);

      const remainingRoles = target.roles.filter(
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
        data: target,
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

  socket.on("ssh", async ({ username, targetIP }) => {
    const users = await getUsers();
    const user = getUserByUsername(username);

    if (!user) {
      socket.emit("sshResult", {
        success: false,
        message: "SSH Connection failed",
        error: "User not found",
      });
      return;
    }

    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const target = internet[targetIP];

    if (!target) {
      socket.emit("sshResult", {
        success: false,
        message: "SSH Connection failed",
        error: "Target IP not found",
      });
      return;
    }

    const sshService = target.services.find(
      (service) => service.name === "ssh"
    );

    if (!sshService) {
      socket.emit("sshResult", {
        success: false,
        message: "SSH Connection failed",
        error: "Service not found",
      });
      return;
    }

    const requiredVulnerabilities = sshService.vulnerabilities.map(
      (vul) => vul.type
    );

    user.exploitedServers = user.exploitedServers || {};
    user.exploitedServers[targetIP] = user.exploitedServers[targetIP] || {};
    const exploitedVulnerabilities = user.exploitedServers[targetIP].ssh || [];
    const exploitedRoles = user.exploitedServers[targetIP].roles || [];

    const allRequiredExploited = requiredVulnerabilities.every((vul) =>
      exploitedVulnerabilities.includes(vul)
    );

    if (!allRequiredExploited) {
      socket.emit("sshResult", {
        success: false,
        message: "SSH Connection failed",
        error: "Prerequisite vulnerabilities not exploited",
      });
      return;
    }

    // Should check user role and permissions here in the future
    // Check if any role is exploited
    const hasExploitedRole = exploitedRoles.length > 0;

    let eventData = null;
    if (hasExploitedRole) {
      eventData = target;
    }

    logAction(username, `Connected to SSH on IP: ${targetIP}`);
    socket.emit("sshResult", {
      success: true,
      message: `Connected to ${targetIP}...`,
      data: eventData,
      targetIP,
      ssh: true,
    });
  });

  socket.on("user_enum", async ({ username, targetIP }) => {
    const users = await getUsers();
    const user = getUserByUsername(username);

    if (!user) {
      socket.emit("userEnumResult", {
        success: false,
        message: "Enumeration failed",
        error: "User not found",
      });
      return;
    }

    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const target = internet[targetIP];

    if (!target) {
      socket.emit("userEnumResult", {
        success: false,
        message: "Enumeration failed",
        error: "Target IP not found",
      });
      return;
    }

    logAction(username, `Enumerated users on IP: ${targetIP}`);
    socket.emit("userEnumResult", {
      success: true,
      message: `User enumeration on ${targetIP} succeeded`,
      data: target.roles,
    });
  });

  socket.on("password_cracker", async ({ username, targetIP, role }) => {
    const users = await getUsers();
    const user = getUserByUsername(username);

    if (!user) {
      socket.emit("passwordCrackerResult", {
        success: false,
        message: "Cracking failed",
        error: "User not found",
      });
      return;
    }

    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const target = internet[targetIP];
    if (!target) {
      socket.emit("passwordCrackerResult", {
        success: false,
        message: "Cracking failed",
        error: "Target IP not found",
      });
      return;
    }

    const sshService = target.services.find(
      (service) => service.name === "ssh"
    );

    if (!sshService) {
      socket.emit("passwordCrackerResult", {
        success: false,
        message: "Cracking failed",
        error: "SSH service not found",
      });
      return;
    }

    const requiredVulnerabilities = sshService.vulnerabilities.map(
      (vul) => vul.type
    );

    const exploitedVulnerabilities = user.exploitedServers[targetIP]?.ssh || [];

    const allRequiredExploited = requiredVulnerabilities.every((vul) =>
      exploitedVulnerabilities.includes(vul)
    );

    if (!allRequiredExploited) {
      socket.emit("passwordCrackerResult", {
        success: false,
        message: "Cracking failed",
        error: "Prerequisite vulnerabilities not exploited",
      });
      return;
    }

    const roleDetails = target.roles.find((r) => r.role === role);

    if (!roleDetails) {
      socket.emit("passwordCrackerResult", {
        success: false,
        message: "Cracking failed",
        error: "Role not found",
      });
      return;
    }

    const tool = user.tools.find((tool) => tool.name === "Password Cracker");
    console.log(tool);
    if (!tool || roleDetails.level > tool.exploits[0].level) {
      socket.emit("passwordCrackerResult", {
        success: false,
        message: "Cracking failed",
        error: "Role level too high for the tool",
      });
      return;
    }

    logAction(username, `Cracked password for role ${role} on IP: ${targetIP}`);
    socket.emit("passwordCrackerResult", {
      success: true,
      message: `Password cracked for role ${role} on ${targetIP}`,
      data: target,
      load: true,
    });
  });

  socket.on("rootkit", async ({ username, targetIP, role }) => {
    const users = await getUsers();
    const user = getUserByUsername(username);

    if (!user) {
      socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "User not found",
      });
      return;
    }

    const internet = await readJSONFile(INTERNET_FILE_PATH);
    const target = internet[targetIP];
    if (!target) {
      socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "Target IP not found",
      });
      return;
    }

    const sshService = target.services.find(
      (service) => service.name === "ssh"
    );

    if (!sshService) {
      socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "SSH service not found",
      });
      return;
    }
    user.exploitedServers[targetIP].roles =
      user.exploitedServers[targetIP].roles || [];

    if (user.exploitedServers[targetIP].roles.includes(role)) {
      socket.emit("rootkitResult", {
        success: true,
        message: "Rootkit already installed",
      });
      return;
    }

    if (!passwordCracked) {
      socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "Password is not cracked",
      });
      return;
    }

    const requiredVulnerabilities = sshService.vulnerabilities.map(
      (vul) => vul.type
    );

    const exploitedVulnerabilities = user.exploitedServers[targetIP]?.ssh || [];

    const allRequiredExploited = requiredVulnerabilities.every((vul) =>
      exploitedVulnerabilities.includes(vul)
    );

    if (!allRequiredExploited) {
      socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "Prerequisite vulnerabilities not exploited",
      });
      return;
    }

    const roleDetails = target.roles.find((r) => r.role === role);

    if (!roleDetails) {
      socket.emit("rootkitResult", {
        success: false,
        message: "Rootkit failed",
        error: "Role not found",
      });
      return;
    }

    // Add a rootkit value instead?
    // Add the role
    if (!user.exploitedServers[targetIP].roles.includes(role)) {
      user.exploitedServers[targetIP].roles.push(role);
    }

    await writeJSONFile(USERS_FILE_PATH, users);

    logAction(
      username,
      `Rootkit initialized for role ${role} on IP: ${targetIP}`
    );
    socket.emit("rootkitResult", {
      success: true,
      message: `Rootkit initialized for role ${role} on ${targetIP}`,
    });
  });
}

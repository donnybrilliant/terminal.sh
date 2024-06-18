// filesystem.js
import { loginManager, socket } from "./index.js";

let fileData = {};
let pathStack = ["home", "users", "guest"]; // Default path for unauthenticated access

let sshFileData = {};
let sshPathStack = [""]; // Default path for SSH

// Function to load the filesystem data from server
async function loadFileSystem() {
  // fun place to do some reboot graphics?
  try {
    const response = await new Promise((resolve) => {
      socket.emit("loadFileSystem", resolve);
    });
    if (response.success) {
      fileData = response.data;
      const username = loginManager.getUsername();
      if (username && username.trim() !== "" && fileData.home.users[username]) {
        pathStack = ["home", "users", username];
      } else {
        pathStack = ["home", "users", "guest"];
        if (!fileData.home.users.guest) {
          fileData.home.users.guest = {
            README: { content: "You are not logged in." },
          };
        }
      }
      return "Filesystem loaded successfully.";
    } else {
      throw new Error(`Error loading filesystem: ${response.message}`);
    }
  } catch (error) {
    throw new Error(`Error loading filesystem: ${error.message}`);
  }
}

// Function to load the target filesystem data for SSH
async function loadTargetFileSystem(targetData) {
  if (!targetData) {
    sshFileData = {};
  } else {
    sshFileData = targetData.fileSystem;
  }
  sshPathStack = [""];
  // pipe this through?
  return "Target filesystem loaded successfully.";
}

// General function to get the current directory
function getCurrentDir(isSSH = false) {
  const currentData = isSSH ? sshFileData : fileData;
  const currentPathStack = isSSH ? sshPathStack : pathStack;

  let dir = currentData;
  for (const part of currentPathStack) {
    if (part && dir[part] && typeof dir[part] === "object") {
      dir = dir[part];
    }
  }

  return dir;
}

// General function to set the current directory
function setCurrentDir(dir, isSSH = false) {
  const currentData = isSSH ? sshFileData : fileData;
  const currentPathStack = isSSH ? sshPathStack : pathStack;

  if (dir.startsWith("/")) {
    // Absolute path
    let tmpDir = currentData; // Start from root
    const parts = dir.split("/").filter(Boolean);

    for (const part of parts) {
      if (part in tmpDir) {
        if (tmpDir[part].content) {
          return "Not a directory";
        }
        tmpDir = tmpDir[part];
      } else {
        return false;
      }
    }

    if (isSSH) {
      sshPathStack = parts.length ? parts : [""];
    } else {
      pathStack = parts.length ? parts : [""];
    }
    return true;
  } else {
    // Relative path
    const currentDir = getCurrentDir(isSSH);

    if (dir === "..") {
      if (currentPathStack.length > 0) {
        currentPathStack.pop();
      }
    } else if (dir in currentDir) {
      if (currentDir[dir].content) {
        return "Not a directory";
      }
      currentPathStack.push(dir);
    } else {
      return false;
    }
    return true;
  }
}

// General function to get the full path as a string
function getCurrentPath(isSSH = false) {
  const currentPathStack = isSSH ? sshPathStack : pathStack;
  const path = currentPathStack.filter(Boolean).join("/");
  return path ? `/${path}` : "/";
}

// General function to get directory names for autocompletion
function getDirectoryNames(isSSH = false) {
  const currentDir = getCurrentDir(isSSH);
  return Object.keys(currentDir).filter(
    (key) => typeof currentDir[key] === "object" && !currentDir[key].content
  );
}

// Function to get file names for autocompletion
function getFileNames(isSSH = false) {
  const currentDir = getCurrentDir(isSSH);
  return Object.keys(currentDir).filter(
    (key) => typeof currentDir[key] === "object" && currentDir[key].content
  );
}

// Function to append a tool to file data
function appendToolToFileData(tool, isSSH = false) {
  const username = loginManager.getUsername();
  const currentData = isSSH ? sshFileData : fileData;
  if (username && username.trim() !== "") {
    if (!currentData.home.users[username].bin) {
      currentData.home.users[username].bin = {};
    }
    currentData.home.users[username].bin[tool.name] = tool;
  }
}

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

// Function to save the user's home directory
async function saveUserHome() {
  const username = loginManager.getUsername();
  if (username && username.trim() !== "" && fileData.home.users[username]) {
    const { README, ...filteredHomeData } = fileData.home.users[username]; // Exclude README

    try {
      const response = await new Promise((resolve) => {
        socket.emit("saveUserHome", filteredHomeData, resolve);
      });

      if (response.success) {
        return response.message;
      } else {
        throw new Error(`Error saving user home: ${response.message}`);
      }
    } catch (error) {
      throw new Error(error.message);
    }
  } else {
    console.log(
      "Skipping save operation: User not recognized or missing home directory."
    );
    return "Skipping save operation: User not recognized or missing home directory.";
  }
}

export {
  getCurrentDir,
  setCurrentDir,
  getCurrentPath,
  getDirectoryNames,
  loadFileSystem,
  loadTargetFileSystem,
  saveUserHome,
  getFileFromPath,
  getFileNames,
  pathStack,
  fileData,
  sshFileData,
  sshPathStack,
  appendToolToFileData,
};

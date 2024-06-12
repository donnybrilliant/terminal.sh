// fileSystem.js
import { loginManager, socket } from "./index.js";

let fileData = {};
let pathStack = ["root", "home", "users", "guest"]; // Default path for unauthenticated access

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
      if (
        username &&
        username.trim() !== "" &&
        fileData.root.home.users[username]
      ) {
        pathStack = ["root", "home", "users", username];
      } else {
        pathStack = ["root", "home", "users", "guest"];
        if (!fileData.root.home.users.guest) {
          fileData.root.home.users.guest = {
            README: "You are not logged in.",
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
  sshFileData = targetData.fileSystem;
  // set startdir in targetDir and set it here.
  sshPathStack = [""];
  // pipe this through?
  return "Target filesystem loaded successfully.";
}

// General function to get the current directory
function getCurrentDir(isSSH = false) {
  const currentData = isSSH ? sshFileData : fileData;
  const currentPathStack = isSSH ? sshPathStack : pathStack;

  // Start from the root
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
      if (part in tmpDir && typeof tmpDir[part] === "object") {
        tmpDir = tmpDir[part];
      } else {
        return false; // Any part in the path is not found or is not a directory
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
    } else if (dir in currentDir && typeof currentDir[dir] === "object") {
      currentPathStack.push(dir);
    } else {
      return false; // Handle error (not a directory or doesn't exist)
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
    (key) => typeof currentDir[key] === "object"
  );
}

// Function to append a tool to file data
function appendToolToFileData(toolName, isSSH = false) {
  const username = loginManager.getUsername();
  const currentData = isSSH ? sshFileData : fileData;
  if (username && username.trim() !== "") {
    if (!currentData.root.home.users[username].bin) {
      currentData.root.home.users[username].bin = {};
    }
    currentData.root.home.users[username].bin[toolName] = toolName;
  }
}

// Function to save the user's home directory
async function saveUserHome() {
  const username = loginManager.getUsername();
  if (
    username &&
    username.trim() !== "" &&
    fileData.root.home.users[username]
  ) {
    const { README, ...filteredHomeData } = fileData.root.home.users[username]; // Exclude README

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

// Export necessary functions
export {
  getCurrentDir,
  setCurrentDir,
  getCurrentPath,
  getDirectoryNames,
  loadFileSystem,
  loadTargetFileSystem,
  saveUserHome,
  pathStack,
  fileData,
  appendToolToFileData,
};

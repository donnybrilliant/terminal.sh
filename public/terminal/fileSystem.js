import { loginManager, socket } from "./index.js";

let fileData = {};
let pathStack = ["root", "home", "users", "guest"]; // Default path for unauthenticated access

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

// Function to get the current directory
function getCurrentDir() {
  return (
    pathStack.reduce(
      (acc, dir) => (acc && acc[dir] ? acc[dir] : undefined),
      fileData
    ) || fileData
  );
}

// Function to set the current directory
function setCurrentDir(dir) {
  if (dir.startsWith("/")) {
    let tmpDir = fileData; // Start from root
    const parts = dir.split("/").filter(Boolean);

    for (const part of parts) {
      if (part in tmpDir && typeof tmpDir[part] === "object") {
        tmpDir = tmpDir[part];
      } else {
        return false; // Any part in the path is not found or is not a directory
      }
    }

    pathStack = parts;
    return true;
  } else {
    const currentDir = getCurrentDir();

    if (dir === "..") {
      if (pathStack.length > 0) {
        pathStack.pop();
      }
    } else if (dir in currentDir && typeof currentDir[dir] === "object") {
      pathStack.push(dir);
    } else {
      return false; // Handle error (not a directory or doesn't exist)
    }
    return true;
  }
}

// Get the full path as a string
function getCurrentPath() {
  return "/" + pathStack.join("/");
}

// Function to get directory names for autocompletion
function getDirectoryNames() {
  const currentDir = getCurrentDir();
  return Object.keys(currentDir).filter(
    (key) => typeof currentDir[key] === "object"
  );
}

function appendToolToFileData(toolName) {
  const username = loginManager.getUsername();
  if (username && username.trim() !== "") {
    if (!fileData.root.home.users[username].bin) {
      fileData.root.home.users[username].bin = {};
    }
    fileData.root.home.users[username].bin[toolName] = toolName;
  }
}

// Export necessary functions
export {
  getCurrentDir,
  setCurrentDir,
  getCurrentPath,
  getDirectoryNames,
  loadFileSystem,
  saveUserHome,
  pathStack,
  fileData,
  appendToolToFileData,
};

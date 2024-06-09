import { loginManager, socket } from "./index.js";

let fileData = {};
let pathStack = ["root", "home", "users", "guest"]; // Default path for unauthenticated access

// Function to load the filesystem data from server
async function loadFileSystem() {
  return new Promise((resolve, reject) => {
    socket.emit("loadFileSystem", (response) => {
      if (response.success) {
        fileData = response.data;
        const username = loginManager.getUsername();
        if (username && fileData.root.home.users[username]) {
          pathStack = ["root", "home", "users", username];
        } else {
          pathStack = ["root", "home", "users", "guest"];
          if (!fileData.root.home.users.guest) {
            fileData.root.home.users.guest = {
              README: "You are not logged in.",
            };
          }
        }
        resolve("Filesystem loaded successfully.");
      } else {
        reject(`Error loading filesystem: ${response.message}`);
      }
    });
  });
}

// Function to save the user's home directory
async function saveUserHome() {
  const username = loginManager.getUsername();
  if (username && fileData.root.home.users[username]) {
    return new Promise((resolve, reject) => {
      const { README, ...filteredHomeData } =
        fileData.root.home.users[username]; // Exclude README

      socket.emit("saveUserHome", filteredHomeData, (response) => {
        if (response.success) {
          resolve(response.message);
        } else {
          reject(`Error saving user home: ${response.message}`);
        }
      });
    });
  } else {
    return Promise.reject("User not recognized or missing home directory.");
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

// Export necessary functions
export {
  getCurrentDir,
  setCurrentDir,
  getCurrentPath,
  loadFileSystem,
  saveUserHome,
};

// Update the user's name
/* function setName(socket, newName, callback) {
  const oldName = loginManager.getUsername(); // Directly use socket.request.user

  if (fileData.root.home.users[newName]) {
    callback("Username already exists. Please choose a different name.");
    return;
  }

  // Send the newName and oldName to the server for updating
  socket.emit("set-name", { oldName, newName }, (response) => {
    if (response.success) {
      updateLocalFileSystemUser(oldName, newName); // Update local filesystem
      callback(`Name updated to ${newName}`, response.user);
    } else {
      callback(`Error updating name: ${response.message}`);
    }
  });
}

function updateLocalFileSystemUser(oldName, newName) {
  fileData.root.home.users[newName] = { ...fileData.root.home.users[oldName] };
  delete fileData.root.home.users[oldName];
  pathStack = ["root", "home", "users", newName];
} */

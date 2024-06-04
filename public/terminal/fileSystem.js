import { fetchWithTimeout } from "../utils/fetch.js";
import { loginManager } from "./index.js";

let fileData = {};
let pathStack = ["root", "home", "users", "guest"]; // Default path for unauthenticated access

// Function to load the filesystem data from server
async function loadFileSystem(apiUrl) {
  try {
    const response = await fetchWithTimeout(`${apiUrl}/filesystem`);
    fileData = response.data; // Load the complete filesystem

    // Setup pathStack based on if the user is logged in
    const username = loginManager.getUsername(); // Assuming this function retrieves the authenticated user's username
    if (username && fileData.root.home.users[username]) {
      pathStack = ["root", "home", "users", username];
    } else {
      pathStack = ["root", "home", "users", "guest"];
      if (!fileData.root.home.users.guest) {
        fileData.root.home.users.guest = { README: "You are not logged in." };
      }
    }

    return "Filesystem loaded successfully.";
  } catch (error) {
    return `Error loading filesystem: ${error.message}`;
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

// Update the user's name
async function setName(newName) {
  const oldName = loginManager.getUsername(); // Fetch the current username from session or a similar method

  if (fileData.root.home.users[newName]) {
    return "Username already exists. Please choose a different name.";
  }

  // Send the newName and oldName to the server for updating
  try {
    const response = await fetchWithTimeout("/set-name", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ oldName, newName }),
    });

    if (response.success) {
      loginManager.setUsername(newName); // Update username in session storage
      updateLocalFileSystemUser(oldName, newName); // Update local filesystem
      return `Name updated to ${newName}`;
    }
  } catch (error) {
    return `Error updating name: ${error.message}`;
  }
}

function updateLocalFileSystemUser(oldName, newName) {
  fileData.root.home.users[newName] = { ...fileData.root.home.users[oldName] };
  delete fileData.root.home.users[oldName];
  pathStack = ["root", "home", "users", newName];
}

async function saveUserHome() {
  const username = loginManager.getUsername(); // Assume we retrieve the current user name from session
  //pathStack.includes(username) instead?
  if (username !== "" && fileData.root.home.users[username]) {
    try {
      const response = await fetchWithTimeout("/update-user-home", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ home: fileData.root.home.users[username] }),
      });

      return response.message;
    } catch (error) {
      return `Error saving user home: ${error.message}`;
    }
  } else {
    return "User not recognized or missing home directory.";
  }
}

export {
  getCurrentDir,
  setCurrentDir,
  getCurrentPath,
  setName,
  loadFileSystem,
  saveUserHome,
};

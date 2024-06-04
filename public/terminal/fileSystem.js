import { fetchWithTimeout } from "../utils/fetch.js";
let promptName = "";
let fileData = {};
let pathStack = ["root", "home", "users", "user"];

// Function to load the filesystem data
async function loadFileSystem(apiUrl) {
  try {
    const data = await fetchWithTimeout(`${apiUrl}/filesystem`);
    populateFileSystem(data.data);
    return "Filesystem loaded successfully.";
  } catch (error) {
    return `Error loading filesystem: ${error.message}`;
  }
}

// Function to get the user's home directory
function getUserHomeDirectory(username) {
  return fileData.root.home.users[username] || {};
}

/**
 * Populate the file system with data.
 * @param {*} data
 */
function populateFileSystem(data, username) {
  console.log(data, username);
  if (data) {
    // Populate with fetched data
    for (const key in data) {
      fileData[key] = data[key];
    }
  }

  if (username) {
    promptName = username;
    pathStack = ["root", "home", "users", username];
    // Ensure the user's home directory exists and has default documents
    if (!fileData.root.home.users[username]) {
      fileData.root.home.users[username] = {
        README: "You are logged in as " + username + ".",
      };
    }
  } else {
    promptName = "";
    pathStack = ["root", "home", "users", "user"];
    // Ensure the default unauthenticated user directory exists and has default documents
    if (!fileData.root.home.users.user) {
      fileData.root.home.users.user = {
        README:
          "You are not logged in. There should be some instructions here.",
      };
    }
  }
}

/**
 * Getter for the current directory.
 *
 * @returns {object} - The current directory.
 */
function getCurrentDir() {
  return (
    pathStack.reduce(
      (acc, dir) => (acc && acc[dir] ? acc[dir] : undefined),
      fileData
    ) || fileData
  );
}

/**
 * Setter for the current directory.
 * Handles movement between directories and maintains a stack for directory traversal.
 *
 * @param {object} dir - The directory to set as current.
 */
function setCurrentDir(dir) {
  if (dir.startsWith("/")) {
    // Handle absolute paths
    let tmpDir = fileData; // Start from root
    const parts = dir.split("/").filter(Boolean); // Get directory parts, removing empty strings

    for (const part of parts) {
      if (part in tmpDir && typeof tmpDir[part] === "object") {
        tmpDir = tmpDir[part];
      } else {
        return false; // Any part in the path is not found or is not a directory
      }
    }

    // Only update pathStack after successfully navigating the path
    pathStack = parts;
    return true;
  } else {
    // Existing logic for relative paths...
    const currentDir = getCurrentDir();

    if (dir === "..") {
      if (pathStack.length > 0) {
        pathStack.pop();
      }
    } else if (dir in currentDir && typeof currentDir[dir] === "object") {
      pathStack.push(dir);
    } else {
      // Handle error (not a directory or doesn't exist)
      return false;
    }
    return true;
  }
}

// Get the full path as a string
function getCurrentPath() {
  return "/" + pathStack.join("/");
}

// Check if the user is within a specific directory
function isWithinDir(directoryName) {
  return pathStack.includes(directoryName);
}
// Get the current user's name - should this be fetched? Or stored on login?
function getName() {
  return promptName;
}

// Update the user's name
async function setName(newName) {
  // Check if user is in the directory being deleted
  const oldName = promptName || "user"; // Take the default "user" if promptName is empty

  // Check if home directory exists
  if (fileData.root && fileData.root.home && fileData.root.home.users) {
    // Check if new username already exists
    if (fileData.root.home.users[newName]) {
      return "Username already exists. Please choose a different name.";
    }

    // Check if old user directory exists
    if (fileData.root.home.users[oldName]) {
      // Duplicate the old user directory to the new name
      fileData.root.home.users[newName] = {
        ...fileData.root.home.users[oldName],
      };

      // Delete the old user directory
      delete fileData.root.home.users[oldName];

      // Update the prompt name
      promptName = newName;

      pathStack = ["root", "home", "users", newName];

      // Update users.json on the server
      try {
        // This assumes the server handles all validation and errors
        await fetchWithTimeout("/set-name", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ oldName, newName }),
        });
        // Update local state only after successful server update
        promptName = newName;
        // Assume pathStack and other related data are also updated appropriately
        return `Name updated to ${newName}`;
      } catch (error) {
        return `Error updating name: ${error.message}`;
      }
    } else {
      return `Home directory not found.`;
    }
  }
}

async function saveUserHome() {
  const username = promptName || "user";
  if (username !== "user" && isWithinDir(username)) {
    try {
      await fetchWithTimeout("/update-user-home", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ home: fileData.root.home.users[promptName] }),
      });
      return "User home updated successfully.";
    } catch (error) {
      return `Error saving user home: ${error.message}`;
    }
  }
}

export {
  getCurrentDir,
  setCurrentDir,
  getCurrentPath,
  getName,
  setName,
  populateFileSystem,
  loadFileSystem,
  getUserHomeDirectory,
  saveUserHome,
};

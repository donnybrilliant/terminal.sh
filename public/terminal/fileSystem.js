let promptName = "";
let fileData = {};

/**
 * Populate the file system with data.
 * @param {*} data
 */
function populateFileSystem(data) {
  if (data) {
    // Populate with fetched data
    for (const key in data) {
      fileData[key] = data[key];
    }
  } else {
    // Fallback: Create default directories and files
    if (!Object.prototype.hasOwnProperty.call(fileData, "home")) {
      fileData.home = {};
    }
    if (!Object.prototype.hasOwnProperty.call(fileData.home, "user")) {
      fileData.home.user = {
        document1: "This is the content of document1.",
        document2: "Another content here for document2.",
      };
    }
  }
}

let pathStack = ["home", "user"]; // We start in /home/user by default

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

function getCurrentPath() {
  return "/" + pathStack.join("/");
}

function isWithinDir(directoryName) {
  const oldName = promptName || "user"; // Take the default "user" if promptName is empty
  return pathStack.includes(oldName) || pathStack.includes(directoryName);
}

function getName() {
  return promptName;
}

function setName(newName) {
  // Check if user is in the directory being deleted
  const oldName = promptName || "user"; // Take the default "user" if promptName is empty
  const inOldUserDir = isWithinDir(oldName);

  if (!newName) {
    return "Please provide a name.";
  }

  // Check if home directory exists
  if (fileData.home) {
    // Check if old user directory exists
    if (fileData.home[oldName]) {
      // Duplicate the old user directory to the new name
      fileData.home[newName] = { ...fileData.home[oldName] };

      // Delete the old user directory
      delete fileData.home[oldName];

      // Update the prompt name
      promptName = newName;

      if (inOldUserDir) {
        // If in the old user directory, navigate to the new one
        pathStack = ["home", newName];
      }

      return `Name updated to ${newName}`;
    } else {
      return `Error: Directory for ${oldName} not found.`;
    }
  } else {
    return "Home directory not found.";
  }
}

export {
  getCurrentDir,
  setCurrentDir,
  getCurrentPath,
  getName,
  setName,
  populateFileSystem,
};

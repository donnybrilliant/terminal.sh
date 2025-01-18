import fs from "fs";
import path from "path";

// Paths to the files and directories we need to modify
const dataDir = "./data";
const messagesDir = path.join(dataDir, "messages");
const usersFilePath = path.join(dataDir, "users.json");
const internetFilePath = path.join(dataDir, "internet.json");
const filesToEmpty = [
  path.join(messagesDir, "general.json"),
  path.join(dataDir, "logs.json"),
];

// Function to overwrite specified files with an empty array
const overwriteWithEmptyArray = (files) => {
  files.forEach((file) => {
    fs.writeFile(file, "[]", (err) => {
      if (err) return console.error(`Error writing to ${file}: ${err}`);
      console.log(`Successfully overwrote ${file} with an empty array.`);
    });
  });
};

// Function to delete all files in the messages directory except for general.json
const cleanMessagesDirectory = (dir) => {
  fs.readdir(dir, (err, files) => {
    if (err) return console.error(`Error reading directory ${dir}: ${err}`);

    files.forEach((file) => {
      if (file !== "general.json") {
        const filePath = path.join(dir, file);
        fs.unlink(filePath, (err) => {
          if (err) return console.error(`Error deleting ${filePath}: ${err}`);
          console.log(`Successfully deleted ${filePath}.`);
        });
      }
    });
  });
};

// Function to remove all users except 'admin' and 'user'
const cleanUsersFile = (filePath) => {
  fs.readFile(filePath, "utf8", (err, data) => {
    if (err) return console.error(`Error reading ${filePath}: ${err}`);

    let users = JSON.parse(data);
    users = users.filter(
      (user) => user.username === "admin" || user.username === "user"
    );

    fs.writeFile(filePath, JSON.stringify(users, null, 2), "utf8", (err) => {
      if (err) return console.error(`Error writing to ${filePath}: ${err}`);
      console.log(`Successfully cleaned ${filePath}.`);
    });
  });
};

// Function to remove 'activeMiners' and 'usedResources' from every server in internet.json
const cleanInternetFile = (filePath) => {
  fs.readFile(filePath, "utf8", (err, data) => {
    if (err) return console.error(`Error reading ${filePath}: ${err}`);

    let internetData = JSON.parse(data);
    Object.keys(internetData).forEach((server) => {
      if (internetData[server].activeMiners !== undefined) {
        delete internetData[server].activeMiners;
      }
      if (internetData[server].usedResources !== undefined) {
        delete internetData[server].usedResources;
      }
    });

    fs.writeFile(
      filePath,
      JSON.stringify(internetData, null, 2),
      "utf8",
      (err) => {
        if (err) return console.error(`Error writing to ${filePath}: ${err}`);
        console.log(`Successfully cleaned ${filePath}.`);
      }
    );
  });
};

// Execute the cleanup functions
overwriteWithEmptyArray(filesToEmpty);
cleanMessagesDirectory(messagesDir);
cleanUsersFile(usersFilePath);
cleanInternetFile(internetFilePath);

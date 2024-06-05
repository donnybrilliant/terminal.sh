import fs from "fs";
import path from "path";

// Paths to the files and directories we need to modify
const dataDir = "./data";
const messagesDir = path.join(dataDir, "messages");
const filesToEmpty = [
  path.join(messagesDir, "general.json"),
  path.join(dataDir, "users.json"),
  path.join(dataDir, "logs.json"), // Adding logs.json to the list
];
const fileSystemPath = path.join(dataDir, "filesystem.json");

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

// Function to modify filesystem.json
const updateFileSystemJson = (filePath) => {
  fs.readFile(filePath, "utf8", (err, data) => {
    if (err) return console.error(`Error reading ${filePath}: ${err}`);
    let jsonData = JSON.parse(data);
    jsonData.root.home.users = [];
    fs.writeFile(filePath, JSON.stringify(jsonData, null, 2), (err) => {
      if (err) return console.error(`Error writing to ${filePath}: ${err}`);
      console.log(`Successfully updated ${filePath}.`);
    });
  });
};

// Execute the cleanup functions
overwriteWithEmptyArray(filesToEmpty);
cleanMessagesDirectory(messagesDir);
updateFileSystemJson(fileSystemPath);

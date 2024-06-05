import { promises as fs } from "fs";
import { fileURLToPath } from "url";
import path, { dirname } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const DATA_DIR = path.join(__dirname, "../data");
const USERS_FILE_PATH = path.join(DATA_DIR, "users.json");
const FILE_SYSTEM_PATH = path.join(DATA_DIR, "filesystem.json");
const LOG_FILE_PATH = path.join(DATA_DIR, "logs.json");

async function readJSONFile(filePath) {
  try {
    const data = await fs.readFile(filePath, "utf-8");
    return JSON.parse(data);
  } catch (err) {
    console.error(`Error reading JSON file at ${filePath}:`, err);
    throw err;
  }
}

async function writeJSONFile(filePath, data) {
  try {
    const jsonData = JSON.stringify(data, null, 2);
    await fs.writeFile(filePath, jsonData);
  } catch (err) {
    console.error(`Error writing JSON file at ${filePath}:`, err);
    throw err;
  }
}

export {
  readJSONFile,
  writeJSONFile,
  USERS_FILE_PATH,
  FILE_SYSTEM_PATH,
  LOG_FILE_PATH,
};

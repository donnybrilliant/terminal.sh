import { promises as fs } from "fs";
import { fileURLToPath } from "url";
import path, { dirname } from "path";
import AsyncLock from "async-lock";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const DATA_DIR = path.join(__dirname, "../data");
const MESSAGES_DIR = path.join(__dirname, "../data/messages");
const USERS_FILE_PATH = path.join(DATA_DIR, "users.json");
const FILE_SYSTEM_PATH = path.join(DATA_DIR, "filesystem.json");
const INTERNET_FILE_PATH = path.join(DATA_DIR, "internet.json");
const TOOLS_FILE_PATH = path.join(DATA_DIR, "tools.json");
const STORE_FILE_PATH = path.join(DATA_DIR, "store.json");
const RESOURCES_FILE_PATH = path.join(DATA_DIR, "resources.json");
const LOG_FILE_PATH = path.join(DATA_DIR, "logs.json");

const lock = new AsyncLock();
const FILE_LOCK_KEY = "file_lock";

async function readJSONFile(filePath) {
  return lock.acquire(FILE_LOCK_KEY, async () => {
    try {
      const data = await fs.readFile(filePath, "utf-8");
      return JSON.parse(data);
    } catch (err) {
      console.error(`Error reading JSON file at ${filePath}:`, err);
      throw err;
    }
  });
}

async function writeJSONFile(filePath, data) {
  return lock.acquire(FILE_LOCK_KEY, async () => {
    try {
      const jsonData = JSON.stringify(data, null, 2);
      await fs.writeFile(filePath, jsonData);
    } catch (err) {
      console.error(`Error writing JSON file at ${filePath}:`, err);
      throw err;
    }
  });
}

export {
  readJSONFile,
  writeJSONFile,
  USERS_FILE_PATH,
  FILE_SYSTEM_PATH,
  LOG_FILE_PATH,
  MESSAGES_DIR,
  INTERNET_FILE_PATH,
  TOOLS_FILE_PATH,
  STORE_FILE_PATH,
  RESOURCES_FILE_PATH,
};

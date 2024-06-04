import { promises as fs } from "fs";
import { fileURLToPath } from "url";
import path, { dirname } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const DATA_DIR = path.join(__dirname, "../data");
const USERS_FILE_PATH = path.join(DATA_DIR, "users.json");
const FILE_SYSTEM_PATH = path.join(DATA_DIR, "filesystem.json");

async function readJSONFile(filePath) {
  const data = await fs.readFile(filePath, "utf-8");
  return JSON.parse(data);
}

async function writeJSONFile(filePath, data) {
  const jsonData = JSON.stringify(data, null, 2);
  await fs.writeFile(filePath, jsonData);
}

export { readJSONFile, writeJSONFile, USERS_FILE_PATH, FILE_SYSTEM_PATH };

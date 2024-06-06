import {
  readJSONFile,
  writeJSONFile,
  LOG_FILE_PATH,
  MESSAGES_DIR,
} from "./fileUtils.js";
import fs from "fs";
import path from "path";

export async function logAction(username, action) {
  const logEntry = { username, action, timestamp: new Date() };

  try {
    const data = await readJSONFile(LOG_FILE_PATH);
    const logs = Array.isArray(data) ? data : [];
    logs.push(logEntry);
    await writeJSONFile(LOG_FILE_PATH, logs);
  } catch (err) {
    console.error("Error logging action:", err);
    // If there's an error (e.g., file doesn't exist), initialize with the log entry
    await writeJSONFile(LOG_FILE_PATH, [logEntry]);
  }
}

export async function logMessage(room, message) {
  const filePath = path.join(MESSAGES_DIR, `${room}.json`);
  let messages = [];
  try {
    if (fs.existsSync(filePath)) {
      messages = await readJSONFile(filePath);
    }
    messages.push(message);
    await writeJSONFile(filePath, messages);
  } catch (err) {
    console.error(`Error logging message to ${filePath}:`, err);
  }
}

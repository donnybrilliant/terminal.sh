import { readJSONFile, writeJSONFile, LOG_FILE_PATH } from "./fileUtils.js";

async function logAction(username, action) {
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

export { logAction };

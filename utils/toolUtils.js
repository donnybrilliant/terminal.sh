import {
  readJSONFile,
  TOOLS_FILE_PATH,
  writeJSONFile,
  INTERNET_FILE_PATH,
} from "./fileUtils.js";

export async function getToolData(toolName) {
  const tools = await readJSONFile(TOOLS_FILE_PATH);
  return tools.tools[toolName];
}

export function mergeTools(currentTool, newTool) {
  const mergedTool = { ...currentTool };

  // Merge resources
  if (newTool.resources) {
    mergedTool.resources = { ...mergedTool.resources };
    for (const key in newTool.resources) {
      mergedTool.resources[key] = Math.min(
        mergedTool.resources[key] || Infinity,
        newTool.resources[key]
      );
    }
  }

  // Merge exploits
  if (newTool.exploits) {
    mergedTool.exploits = mergedTool.exploits || [];
    newTool.exploits.forEach((newExploit) => {
      const existingExploit = mergedTool.exploits.find(
        (exploit) => exploit.type === newExploit.type
      );
      if (existingExploit) {
        existingExploit.level = Math.max(
          existingExploit.level || 0,
          newExploit.level
        );
      } else {
        mergedTool.exploits.push(newExploit);
      }
    });
  }

  // Merge other properties if they exist in the new tool
  if (newTool.function) {
    mergedTool.function = newTool.function;
  }

  return mergedTool;
}

export function getFileFromPath(fileSystem, filePath) {
  const pathParts = filePath.split("/").filter(Boolean); // Filter out empty parts
  let currentDir = fileSystem;

  for (const part of pathParts) {
    if (currentDir.hasOwnProperty(part)) {
      currentDir = currentDir[part];
    } else {
      console.log(`Path part not found: ${part}`);
      return null;
    }
  }

  return currentDir;
}

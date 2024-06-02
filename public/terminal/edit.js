import { getCurrentDir } from "./fileSystem.js";

let currentEditingFile = null;
let originalFileContent = "";
let inEditMode = false;
let editedContent = "";

export function editFile(filename) {
  const currentDir = getCurrentDir();

  if (!filename) {
    return "Please specify a file to edit.";
  }

  if (!(filename in currentDir)) {
    return `Error: ${filename} does not exist. Create it first using 'touch'.`;
  }

  currentEditingFile = filename;
  originalFileContent = currentDir[filename];
  inEditMode = true;

  // Return the content to be edited
  editedContent = originalFileContent;
  return originalFileContent;
  //return currentDir[filename];
  //return `Editing ${filename}\n\n${originalFileContent}\n\n[Type your additions or changes below. Type ':save' to save and ':exit' to exit without saving.]`;
}

export function saveEdits() {
  if (!inEditMode || !currentEditingFile) {
    return "No file is currently being edited.";
  }

  const currentDir = getCurrentDir();
  currentDir[currentEditingFile] = editedContent.replace(":save", "").trim();

  inEditMode = false;
  currentEditingFile = null;
  originalFileContent = "";
  console.log(editedContent);

  editedContent = ""; // reset the edited content

  return "Changes saved.";
}

export function exitEdit() {
  if (!inEditMode) {
    return "No file is currently being edited.";
  }

  inEditMode = false;
  currentEditingFile = null;
  originalFileContent = "";

  return "Exited without saving changes.";
}

export function isInEditMode() {
  return inEditMode;
}

export function updateFileContent(newContent) {
  originalFileContent += newContent;
}

export function appendToEditedContent(input) {
  if (input === "") {
    editedContent += "\n"; // Handle Enter key by appending a newline
  } else {
    if (!editedContent && input) {
      editedContent = input; // Initialize editedContent if it's empty
    } else {
      editedContent += "\n" + input; // Add a newline before the new input
    }
  }
}

export function getEditedContent() {
  return editedContent;
}

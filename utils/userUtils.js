import { readJSONFile, writeJSONFile, USERS_FILE_PATH } from "./fileUtils.js";

let users = [];
let onlineUsers = new Set();
let guestCount = 0;

export async function loadUsers() {
  try {
    users = await readJSONFile(USERS_FILE_PATH);
    users.forEach((user) => {
      user.username = String(user.username); // Ensure usernames are strings
    });
  } catch (err) {
    console.error("Error loading users:", err);
    users = [];
  }
}

export async function saveUsers() {
  try {
    await writeJSONFile(USERS_FILE_PATH, users);
  } catch (err) {
    console.error("Error saving users:", err);
  }
}

export async function getUsers() {
  await loadUsers();
  return users;
}

export function setUsers(newUsers) {
  users = newUsers;
}

export function getOnlineUsers() {
  return onlineUsers;
}

export function getGuestCount() {
  return guestCount;
}

export function incrementGuestCount() {
  guestCount++;
}

export function decrementGuestCount() {
  guestCount--;
}

export function addOnlineUser(username) {
  onlineUsers.add(username);
}

export function removeOnlineUser(username) {
  onlineUsers.delete(username);
}

export function findUserSocket(username, namespace) {
  const sockets = namespace.sockets;
  for (let [id, socket] of sockets) {
    if (socket.username === username) {
      return socket;
    }
  }
  return null;
}

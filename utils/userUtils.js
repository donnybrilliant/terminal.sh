import {
  readJSONFile,
  writeJSONFile,
  USERS_FILE_PATH,
  INTERNET_FILE_PATH,
} from "./fileUtils.js";

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

export async function saveUser(user) {
  try {
    const userIndex = users.findIndex((u) => u.username === user.username);
    if (userIndex >= 0) {
      users[userIndex] = user;
      await writeJSONFile(USERS_FILE_PATH, users);
    }
  } catch (err) {
    console.error("Error saving user:", err);
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

export function getUserByUsername(username) {
  return users.find((user) => user.username === username);
}

export async function checkUser(username) {
  const users = await getUsers();
  const user = getUserByUsername(username);
  return { users, user };
}

export async function checkTargetIP(targetIP, parentIP = null) {
  const internet = await readJSONFile(INTERNET_FILE_PATH);

  // If parentIP is provided, search within its localNetwork
  if (parentIP && internet[parentIP] && internet[parentIP].localNetwork) {
    if (internet[parentIP].localNetwork[targetIP]) {
      return internet[parentIP].localNetwork[targetIP];
    }
  }

  // Check if targetIP is in the main internet structure
  if (internet[targetIP]) {
    return internet[targetIP];
  }

  return null;
}

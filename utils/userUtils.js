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

  // Helper function to recursively find the server and construct the path
  const findServer = (network, ip, path) => {
    if (network[ip]) {
      // Append the current IP to the path if found and return the current network and the path
      return {
        server: network[ip],
        path: path.length > 0 ? path + ".localNetwork." + ip : ip,
      };
    }
    // Loop through each network key looking for nested localNetworks
    for (const key in network) {
      if (network[key].localNetwork) {
        const newPath = path.length > 0 ? path + ".localNetwork." + key : key;
        const result = findServer(network[key].localNetwork, ip, newPath);
        if (result) return result; // Return as soon as a match is found
      }
    }
    return null;
  };

  if (parentIP) {
    // Start from the parent network and construct the path from there
    const parentResult = findServer(internet, parentIP, "");
    if (parentResult && parentResult.server.localNetwork) {
      return findServer(
        parentResult.server.localNetwork,
        targetIP,
        parentResult.path
      );
    }
  } else {
    // If no parentIP is specified, start from the root of the internet
    return findServer(internet, targetIP, "");
  }
}

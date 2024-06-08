import {
  getUsers,
  setUsers,
  saveUsers,
  findUserSocket,
} from "../utils/userUtils.js";

export function setupAllianceHandlers(socket, chatNamespace) {
  socket.on("createAlliance", async (data) => {
    let { usernames, creator } = data;

    if (creator.toLowerCase() === "guest") {
      socket.emit("message", "Guests cannot create alliances. Please log in.");
      return;
    }

    if (!usernames.includes(creator)) {
      usernames.push(creator);
    }

    // Sort the usernames
    usernames.sort();

    const users = await getUsers();

    // Check if all provided usernames exist in users.json
    const invalidUsernames = usernames.filter(
      (username) => !users.some((u) => u.username === username)
    );

    if (invalidUsernames.length > 0) {
      socket.emit(
        "message",
        `The following usernames do not exist: ${invalidUsernames.join(", ")}`
      );
      return;
    }

    const allianceRoom = `alliance-${usernames.join("-")}`;

    for (const username of usernames) {
      const user = users.find((u) => u.username === username);
      if (user) {
        if (!user.alliance) {
          user.alliance = [];
        }
        // Check if the allianceRoom already exists
        if (!user.alliance.includes(allianceRoom)) {
          user.alliance.push(allianceRoom);
        } else {
          // If the allianceRoom already exists, return or throw an error
          return;
        }
      }
    }

    setUsers(users); // Update the users array
    await saveUsers();

    usernames.forEach((username) => {
      const userSocket = findUserSocket(username, chatNamespace);
      if (userSocket && username !== creator) {
        userSocket.emit(
          "message",
          `You are added to the alliance '${allianceRoom}'. Use ':join ${allianceRoom}' to join.`
        );
      }
    });

    const creatorSocket = findUserSocket(creator, chatNamespace);
    if (creatorSocket) {
      creatorSocket.leave("general");
      creatorSocket.join(allianceRoom);
      creatorSocket.currentRoom = allianceRoom;
      creatorSocket.emit(
        "message",
        `You have been moved to the new alliance room: ${allianceRoom}`
      );
      creatorSocket.emit("roomChanged", allianceRoom); // Notify client of the room change
    }
  });

  socket.on("listAlliances", async () => {
    const users = await getUsers();
    const user = users.find((u) => u.username === socket.username);
    if (user && user.alliance) {
      socket.emit("listAlliances", user.alliance);
    } else {
      socket.emit("listAlliances", []);
    }
  });
}

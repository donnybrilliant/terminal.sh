import {
  getUsers,
  setUsers,
  saveUsers,
  findUserSocket,
} from "../utils/userUtils.js";

export function setupAllianceHandlers(socket, chatNamespace) {
  socket.on("createAlliance", async (data) => {
    let { usernames, creator } = data;
    const allianceRoom = `alliance-${creator}-${Date.now()}`;

    if (!usernames.includes(creator)) {
      usernames.push(creator);
    }

    const users = await getUsers();
    for (const username of usernames) {
      const user = users.find((u) => u.username === username);
      if (user) {
        if (!user.alliance) {
          user.alliance = [];
        }
        user.alliance.push(allianceRoom);
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
    }
  });

  socket.on("listAlliances", async () => {
    const users = await getUsers();
    const user = users.find((u) => u.username === socket.username);
    if (user && user.alliance) {
      socket.emit("message", `Your alliances: ${user.alliance.join(", ")}`);
    } else {
      socket.emit("message", "You have no alliances.");
    }
  });
}

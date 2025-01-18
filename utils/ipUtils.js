export function generateUniqueIP(users, internet) {
  const usedIPs = new Set(users.map((user) => user.ip));
  Object.keys(internet).forEach((ip) => usedIPs.add(ip));

  let ip;
  do {
    ip = Array(4)
      .fill(0)
      .map(() => Math.floor(Math.random() * 256))
      .join(".");
  } while (usedIPs.has(ip) || isPrivateIP(ip));
  return ip;
}

export function generateLocalIP() {
  const parts = [
    10,
    Math.floor(Math.random() * 256),
    Math.floor(Math.random() * 256),
    Math.floor(Math.random() * 256),
  ];
  return parts.join(".");
}

export function generateLocalNetworkIP(baseIP, usedIPs) {
  const baseParts = baseIP.split(".").map(Number);
  let ip;
  do {
    ip = [
      baseParts[0],
      Math.floor(Math.random() * 256),
      Math.floor(Math.random() * 256),
      Math.floor(Math.random() * 256),
    ].join(".");
  } while (usedIPs.has(ip));
  return ip;
}

export function isPrivateIP(ip) {
  const parts = ip.split(".").map(Number);
  return (
    parts[0] === 10 ||
    (parts[0] === 172 && parts[1] >= 16 && parts[1] <= 31) ||
    (parts[0] === 192 && parts[1] === 168)
  );
}

export function generateUniqueMAC(users, internet) {
  const usedMACs = new Set(users.map((user) => user.mac));
  Object.values(internet).forEach((server) => usedMACs.add(server.mac));

  let mac;
  do {
    mac = Array(6)
      .fill(0)
      .map(() =>
        ("00" + Math.floor(Math.random() * 256).toString(16)).slice(-2)
      )
      .join(":");
  } while (usedMACs.has(mac));
  return mac;
}

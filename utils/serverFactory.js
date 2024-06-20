// serverFactory.js

import { v4 as uuidv4 } from "uuid";
import {
  generateUniqueIP,
  generateLocalIP,
  generateLocalNetworkIP,
  generateUniqueMAC,
} from "../utils/ipUtils.js";

export async function createServer(
  config,
  internet = {},
  users = [],
  baseLocalIP = null,
  isLocal = false
) {
  const {
    ip = null,
    localIP = null,
    mac = null,
    securityLevel = 1,
    resources = { cpu: 2000, bandwidth: 500, ram: 8 },
    wallet = { crypto: 0.5, data: 1000 },
    tools = [],
    services = [],
    roles = [],
    logs = [],
    fileSystem = {},
    connectedIPs = [],
  } = config;

  const finalIP = isLocal ? null : generateUniqueIP(users, internet);
  const finalLocalIP =
    localIP ||
    (baseLocalIP
      ? generateLocalNetworkIP(
          baseLocalIP,
          new Set([
            ...Object.keys(internet),
            ...Object.values(internet).flatMap((server) =>
              Object.keys(server.localNetwork || {})
            ),
          ])
        )
      : generateLocalIP());
  const finalMAC = mac || generateUniqueMAC(users, internet);

  return {
    id: uuidv4(),
    ip: finalIP,
    localIP: finalLocalIP,
    mac: finalMAC,
    securityLevel,
    resources,
    wallet,
    tools,
    services,
    roles,
    logs,
    fileSystem,
    connectedIPs,
  };
}

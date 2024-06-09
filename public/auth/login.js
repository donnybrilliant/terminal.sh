import { fetchWithTimeout } from "../utils/fetch.js";

export class LoginManager {
  constructor(socket, apiUrl) {
    this.socket = socket;
    this.apiUrl = apiUrl;
    this.username = null;
  }

  setTerminal(term) {
    this.term = term;
  }

  setUsername(username) {
    this.username = username;
  }

  getUsername() {
    return this.username;
  }

  async initializeLoginState() {
    this.socket.connect(); // Connect as a guest initially
    const token = localStorage.getItem("jwtToken");
    if (token) {
      await this.authenticateSocket(token);
    }
  }

  async authenticateSocket(token) {
    const response = await this.emitSocketEvent("authenticate", token);
    if (response.success) {
      this.setUsername(response.user.username);
      console.log(`Authenticated as ${response.user.username}`);
    } else {
      console.log(response.message);
      localStorage.removeItem("jwtToken");
      throw new Error(response.message);
    }
    // load file system?
  }

  emitSocketEvent(event, data) {
    return new Promise((resolve, reject) => {
      this.socket.emit(event, data, (response) => {
        if (response) {
          resolve(response);
        } else {
          reject(new Error("No response from socket event"));
        }
      });
    });
  }

  async login(username, password) {
    const token = localStorage.getItem("jwtToken");
    if (token) {
      this.term.write(`\r\nAlready logged in.\r\n`);
      return;
    }

    try {
      const result = await fetchWithTimeout(`${this.apiUrl}/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ username, password }),
      });

      if (result.success) {
        const { token, user } = result.data;
        localStorage.setItem("jwtToken", token);
        await this.authenticateSocket(token);
        this.term.write(`\r\n${result.message}\r\n`);
      } else {
        this.term.write(`\r\n${result.message}\r\n`);
      }
    } catch (error) {
      console.error(`Failed to log in: ${error.message}`);
      this.term.write(`\r\nFailed to log in: ${error.message}\r\n`);
    }
  }

  async logout() {
    localStorage.removeItem("jwtToken");
    await this.emitSocketEvent("authenticate", null);
    this.socket.auth = {};
    this.setUsername(null);
    this.term.write(`\r\nLogged out successfully.\r\n`);
    this.socket.disconnect();
    await this.initializeLoginState();
  }
}

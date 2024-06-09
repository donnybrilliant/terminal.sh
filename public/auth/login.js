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
    this.socket.connect();
    const token = localStorage.getItem("jwtToken");
    if (token) {
      this.authenticateSocket(token);
    }
  }

  async authenticateSocket(token) {
    this.socket.emit("authenticate", token, (response) => {
      if (response.success) {
        this.setUsername(response.user.username);
        console.log(`Authenticated as ${response.user.username}`);
      } else {
        console.log(response.message);
        localStorage.removeItem("jwtToken");
      }
    });
  }

  async login(username, password) {
    const token = localStorage.getItem("jwtToken");
    if (token) {
      this.term.write(`\r\nAlready logged in.\r\n`);
      return; // Exit if already logged in
    }

    try {
      const result = await fetchWithTimeout(`${this.apiUrl}/login`, {
        method: "POST",
        body: JSON.stringify({ username, password }),
      });
      console.log(result);
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
    this.socket.emit("authenticate", null, () => {
      this.socket.auth = {};
      this.setUsername(null);
      this.term.write(`\r\nLogged out successfully.\r\n`);
      this.socket.disconnect();
    });
    await this.initializeLoginState();
  }
}

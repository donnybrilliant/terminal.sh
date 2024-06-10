import { loadFileSystem, pathStack, fileData } from "../terminal/fileSystem.js";
import { fetchWithTimeout } from "../utils/fetch.js";

export class LoginManager {
  constructor(socket, apiUrl) {
    this.socket = socket;
    this.apiUrl = apiUrl;
    this.username = "";
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
    this.socket.on("connect", async () => {
      const token = localStorage.getItem("jwtToken");
      if (token) {
        await this.authenticateSocket(token);
      } else {
        await loadFileSystem();
      }
    });
  }

  async authenticateSocket(token) {
    try {
      const response = await new Promise((resolve) => {
        this.socket.emit("authenticate", token, resolve);
      });
      if (response.success) {
        this.setUsername(response.user.username);
      } else {
        console.log(response.message);
        localStorage.removeItem("jwtToken");
        this.setUsername("");
      }
      await loadFileSystem();
    } catch (error) {
      console.error(`Authentication error: ${error.message}`);
    }
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
    this.socket.emit("authenticate", null, async () => {
      this.socket.auth = {};
      this.setUsername("");
      this.socket.disconnect();
      await this.initializeLoginState();
      this.term.write(`\r\nLogged out successfully.\r\n`);
    });
  }
  async checkAuth() {
    try {
      const token = localStorage.getItem("jwtToken");
      const localUsername = this.getUsername();
      if (!token || !localUsername) {
        // No JWT token or username, consider as not authenticated
        return false;
      }
      const isAuthenticated = await new Promise((resolve) => {
        this.socket.emit("check-auth");
        this.socket.on("auth-status", (data) => {
          resolve(data.authenticated);
        });
      });
      return isAuthenticated;
    } catch (error) {
      console.error(`Authentication check error: ${error.message}`);
      return false;
    }
  }
  async setName(newName) {
    const oldName = this.getUsername();
    if (!oldName) {
      this.term.write(`\r\nError updating name: No user logged in\r\n`);
      return;
    }

    if (fileData.root.home.users[newName]) {
      this.term.write(
        `\r\nError updating name: Username ${newName} already exists\r\n`
      );
      return;
    }

    if (!fileData.root.home.users[oldName]) {
      this.term.write(
        `\r\nError updating name: Username ${oldName} not found\r\n`
      );
      return;
    }

    this.socket.emit("setName", { oldName, newName }, (response) => {
      if (response.success) {
        this.setUsername(newName);
        try {
          fileData.root.home.users[newName] = {
            ...fileData.root.home.users[oldName],
          };
          delete fileData.root.home.users[oldName];
          pathStack.length = 0;
          pathStack.push("root", "home", "users", newName);
          this.term.write(`\r\nName updated to ${newName}\r\n`);
        } catch (error) {
          this.term.write(
            `\r\nError updating local filesystem: ${error.message}\r\n`
          );
        }
      } else {
        this.term.write(`\r\nError updating name: ${response.message}\r\n`);
      }
    });
  }
}

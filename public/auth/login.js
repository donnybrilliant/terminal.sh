import { loadFileSystem } from "../terminal/fileSystem.js";
import { fetchWithTimeout } from "../utils/fetch.js";

export class LoginManager {
  constructor(apiUrl) {
    this.apiUrl = apiUrl;
  }

  setTerminal(term) {
    this.term = term;
  }

  setUsername(username) {
    sessionStorage.setItem("username", username);
  }

  getUsername() {
    return sessionStorage.getItem("username") || "";
  }

  clearUsername() {
    sessionStorage.removeItem("username");
  }

  async initializeLoginState() {
    try {
      const status = await this.checkAuthStatus();
      if (status.data.authenticated) {
        this.setUsername(status.data.user.username);
        await loadFileSystem(this.apiUrl);
        // return this?
        console.log(`Logged in as ${status.data.user.username}`);
      } else {
        await loadFileSystem(this.apiUrl);
      }
    } catch (error) {
      console.log(`Failed to check login status: ${error.message}`);
    }
  }

  async login(username, password) {
    const status = await this.checkAuthStatus();
    if (status.data.authenticated) {
      this.term.write(
        `\r\nUser already logged in as ${status.data.user.username}\r\n`
      );
    } else {
      const result = await this.authenticateUser(username, password);
      this.term.write(`\r\n${result.message}\r\n`);
    }
  }

  async authenticateUser(username, password) {
    try {
      const data = await fetchWithTimeout(`${this.apiUrl}/login`, {
        method: "POST",
        body: JSON.stringify({ username, password }),
      });
      if (data.success) {
        this.setUsername(username);
        await loadFileSystem(this.apiUrl);
      }
      return data;
    } catch (error) {
      return error;
    }
  }

  async logout() {
    try {
      const status = await this.checkAuthStatus();
      if (status.data.authenticated) {
        const data = await fetchWithTimeout(`${this.apiUrl}/logout`, {
          method: "POST",
        });
        this.clearUsername();
        await loadFileSystem(this.apiUrl);
        this.term.write(`\r\n${data.message}\r\n`);
      } else {
        this.term.write(`\r\nYou are not logged in.`);
      }
    } catch (error) {
      this.term.write(`\r\nError logging out: ${error.message}\r\n`);
    }
  }

  async checkAuthStatus() {
    try {
      return await fetchWithTimeout(`${this.apiUrl}/auth-status`);
    } catch (error) {
      this.term.write(
        `\r\nError checking authentication status: ${error.message}\r\n`
      );
      throw error;
    }
  }
}

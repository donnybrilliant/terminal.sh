import { loadFileSystem, populateFileSystem } from "./fileSystem.js";
import { fetchWithTimeout } from "../utils/fetch.js";

export class LoginManager {
  constructor(apiUrl) {
    this.apiUrl = apiUrl;
  }

  setTerminal(term) {
    this.term = term;
  }

  async initializeLoginState() {
    try {
      const status = await this.checkAuthStatus();
      if (status.data.authenticated) {
        await this.fetchFileSystem(this.apiUrl, status.data.user.username);
        return `\r\nLogged in as ${status.data.user.username}\r\n${status.data.user.username}$ `;
      } else {
        await loadFileSystem(this.apiUrl);
      }
    } catch (error) {
      return `\r\nFailed to check login status: ${error.message}\r\n$ `;
    }
  }

  async login(username, password) {
    try {
      const status = await this.checkAuthStatus();
      if (status.data.authenticated) {
        this.term.write(
          `\r\nUser already logged in as ${status.data.user.username}\r\n$ `
        );
      } else {
        await this.authenticateUser(username, password);
      }
    } catch (error) {
      this.term.write(
        `\r\nError checking authentication status: ${error.message}\r\n$ `
      );
    }
  }

  async authenticateUser(username, password) {
    try {
      const data = await fetchWithTimeout(`${this.apiUrl}/login`, {
        method: "POST",
        body: JSON.stringify({ username, password }),
      });
      await this.fetchFileSystem(this.apiUrl, username);
      this.term.write(`\r\n${data.message}\r\n${username}$ `);
    } catch (error) {
      this.term.write(`\r\nError logging in: ${error.message}\r\n$ `);
    }
  }

  async fetchFileSystem(apiUrl, username) {
    try {
      const data = await fetchWithTimeout(`${apiUrl}/filesystem`);
      populateFileSystem(data.data, username);
    } catch (error) {
      this.term.write(`\r\nError fetching file system: ${error.message}\r\n$ `);
    }
  }

  async logout() {
    try {
      const status = await this.checkAuthStatus();
      if (status.data.authenticated) {
        const data = await fetchWithTimeout(`${this.apiUrl}/logout`, {
          method: "POST",
        });
        this.term.write(`\r\n${data.message}\r\n$ `);
        await loadFileSystem(this.apiUrl);
      } else {
        this.term.write(`\r\nYou are not logged in.\r\n$ `);
      }
    } catch (error) {
      this.term.write(`\r\nError logging out: ${error.message}\r\n$ `);
    }
  }

  async checkAuthStatus() {
    try {
      return await fetchWithTimeout(`${this.apiUrl}/auth-status`);
    } catch (error) {
      this.term.write(
        `\r\nError checking authentication status: ${error.message}\r\n$ `
      );
      throw error;
    }
  }
}

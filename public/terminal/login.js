import { populateFileSystem } from "./fileSystem.js";
export class LoginManager {
  constructor(apiUrl) {
    this.apiUrl = apiUrl;
  }

  setTerminal(term) {
    this.term = term;
  }

  async login(username, password) {
    try {
      const status = await this.checkAuthStatus();
      if (status.authenticated) {
        this.term.write(
          `\r\nUser already logged in as ${status.user.username}\r\n$ `
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
      const response = await fetch(`${this.apiUrl}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });
      const data = await response.json(); // Parse JSON only once
      if (!response.ok) {
        throw new Error(
          data.message || `HTTP error! status: ${response.status}`
        );
      }
      if (data.success) {
        await this.fetchFileSystem(this.apiUrl, data.user.username);
        this.term.write(`\r\n${data.message}\r\n$ `); // Use server message directly
      } else {
        this.term.write(`\r\n${data.message}\r\n$ `);
      }
    } catch (error) {
      this.term.write(`\r\nError logging in: ${error.message}\r\n$ `);
    }
  }

  async fetchFileSystem(apiUrl, username) {
    try {
      const response = await fetch(`${apiUrl}/filesystem`);
      const data = await response.json();
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      populateFileSystem(data, username);
    } catch (error) {
      this.term.write(`\r\nError fetching file system: ${error.message}\r\n$ `);
    }
  }

  async logout() {
    try {
      const response = await fetch(`${this.apiUrl}/logout`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      const data = await response.json();
      if (!response.ok) {
        throw new Error(
          data.message || `HTTP error! status: ${response.status}`
        );
      }
      this.term.write(`\r\n${data.message}\r\n$ `);
    } catch (error) {
      this.term.write(`\r\nError logging out: ${error.message}\r\n$ `);
    }
  }

  async checkAuthStatus() {
    try {
      const response = await fetch(`${this.apiUrl}/auth-status`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      const data = await response.json();
      if (!response.ok) {
        throw new Error(
          data.message || `HTTP error! status: ${response.status}`
        );
      }
      return data;
    } catch (error) {
      this.term.write(
        `\r\nError checking authentication status: ${error.message}\r\n$ `
      );
      throw error; // Optionally rethrow to handle it outside if needed
    }
  }
}

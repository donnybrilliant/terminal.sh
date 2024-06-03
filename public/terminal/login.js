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
      console.error("Auth Status Check Error:", error);
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
      if (!response.ok) {
        const data = await response.json();
        throw new Error(
          data.message || `HTTP error! status: ${response.status}`
        );
      }
      const data = await response.json();
      if (data.success) {
        await this.fetchFileSystem(this.apiUrl, data.user.username);
        this.term.write(
          `\r\nLogin successful! Welcome ${data.user.username}\r\n$ `
        );
      } else {
        this.term.write(`\r\n${data.message}\r\n$ `);
      }
    } catch (error) {
      console.error("Login Error:", error);
      this.term.write(`\r\nError logging in: ${error.message}\r\n$ `);
    }
  }

  async fetchFileSystem(apiUrl, username) {
    try {
      const response = await fetch(`${apiUrl}/filesystem`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      populateFileSystem(data, username);
    } catch (error) {
      console.error("Fetch File System Error:", error);
      this.term.write(`\r\nError fetching file system: ${error.message}\r\n$ `);
    }
  }

  logout() {
    fetch(`${this.apiUrl}/logout`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
      })
      .then((data) => {
        this.term.write(`\r\n${data.message}\r\n$ `);
      })
      .catch((error) => {
        console.error("Logout Error:", error);
        this.term.write(`\r\nError logging out: ${error.message}\r\n$ `);
      });
  }

  checkAuthStatus() {
    return fetch(`${this.apiUrl}/auth-status`, {
      method: "GET",
      headers: { "Content-Type": "application/json" },
    }).then((response) => {
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return response.json();
    });
  }
}

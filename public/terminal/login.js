// login.js
export class LoginManager {
  constructor(apiUrl) {
    this.apiUrl = apiUrl;
  }

  setTerminal(term) {
    this.term = term;
  }

  login(username, password) {
    this.authenticateUser(username, password);
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

  authenticateUser(username, password) {
    fetch(`${this.apiUrl}/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
      })
      .then((data) => {
        if (data.success) {
          this.term.write(
            `\r\nLogin successful! Welcome ${data.user.username}\r\n$ `
          );
        } else {
          this.term.write("\r\nLogin failed. Try again\r\n$ ");
        }
      })
      .catch((error) => {
        console.error("Login Error:", error);
        this.term.write(`\r\nError logging in: ${error.message}\r\n$ `);
      });
  }
}

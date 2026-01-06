# terminal.sh Server

A terminal-based hacking game server built in Go, featuring a virtual filesystem, network exploration, and tool-based gameplay. Available via both SSH and WebSocket (browser) interfaces.

## Quick Start

For development, simply run:

```bash
make dev
```

This will build all binaries and start the combined server (SSH on port 2222, Web on port 8080).

See the [Building](#building) and [Running](#running) sections below for more details.

## Documentation

- **[GAMEPLAY.md](GAMEPLAY.md)** - Complete gameplay guide with commands, strategies, and tips
- **README.md** (this file) - Technical documentation for building, running, and deploying

## Architecture

The project supports three deployment options:

- **Combined Server** (`cmd/all`) - Runs both SSH and WebSocket servers together (recommended for single-machine deployments)
- **SSH Server** (`cmd/ssh`) - Traditional SSH terminal interface only
- **Web Server** (`cmd/web`) - HTTP/WebSocket server for browser access only

All servers share the same core game logic and can use the same database. The combined server is the easiest way to get started, while separate servers allow for independent scaling and deployment.

## Building

Binaries are built to the `bin/` directory (which is gitignored). The project includes a `Makefile` for convenient build and run commands.

### Quick Start (Development)

For development, use the `dev` command to build and start the combined server:

```bash
make dev
```

This will:

1. Build all binaries (SSH, Web, and Combined)
2. Start the combined server (SSH on port 2222, Web on port 8080)

### Using Makefile

The project includes a `Makefile` with convenient commands:

```bash
# Build all binaries
make build          # or: make build-all

# Build specific binaries
make build-ssh      # SSH server only
make build-web      # Web server only

# Build and run
make dev            # Build all + start combined server (recommended for dev)
make run            # Build all + start combined server (alias for run-all)
make run-ssh        # Build and run SSH server only
make run-web        # Build and run Web server only

# Clean built binaries
make clean

# Show all available commands
make help
```

### Manual Building

If you prefer to build manually:

**Build SSH Server:**

```bash
mkdir -p bin
go build -o bin/terminal.sh-ssh ./cmd/ssh
```

**Build Web Server:**

```bash
go build -o bin/terminal.sh-web ./cmd/web
```

**Build Combined Server:**

```bash
go build -o bin/terminal.sh ./cmd/all
```

**Build All:**

```bash
mkdir -p bin
go build -o bin/terminal.sh-ssh ./cmd/ssh
go build -o bin/terminal.sh-web ./cmd/web
go build -o bin/terminal.sh ./cmd/all
```

## Running

### Development Mode (Recommended)

For development, use the `dev` command to build and start everything:

```bash
make dev
```

This will:

1. Build all binaries
2. Start the combined server (SSH + Web)

The server will start:

- SSH server on `0.0.0.0:2222` (default)
- Web server on `0.0.0.0:8080` (default)

### Combined Server (Manual)

If you've already built the binaries, you can run directly:

```bash
./bin/terminal.sh
```

Or using Makefile:

```bash
make run        # Builds and runs combined server
```

This will start:

- SSH server on `0.0.0.0:2222` (default)
- Web server on `0.0.0.0:8080` (default)

### SSH Server Only

```bash
./bin/terminal.sh-ssh
```

Or using Makefile:

```bash
make run-ssh    # Builds and runs SSH server only
```

The SSH server will listen on `0.0.0.0:2222` by default.

### Web Server Only

```bash
./bin/terminal.sh-web
```

Or using Makefile:

```bash
make run-web    # Builds and runs Web server only
```

The web server will listen on `0.0.0.0:8080` by default.

### Running Both Separately

You can also run both servers in separate terminals:

```bash
# Terminal 1
make run-ssh    # or: ./bin/terminal.sh-ssh

# Terminal 2
make run-web    # or: ./bin/terminal.sh-web
```

## Configuration

Configuration can be set via:

1. **Environment variables** (production/Docker - recommended)
2. **`.env` file** (local development - optional, gitignored)

Environment variables take precedence over `.env` file values. For local development, create a `.env` file from `.env.example`:

```bash
cp .env.example .env
# Edit .env with your settings
```

### Environment Variables:

### SSH Server

- `HOST` - Host to bind to (default: `0.0.0.0`)
- `PORT` - Port to listen on (default: `2222`)
- `HOSTKEY_PATH` - Path to SSH host key file (optional)
  - If not provided, defaults to `.ssh/ssh_host_key`
  - The Wish framework will automatically generate a new host key if the file doesn't exist
  - **Important**: The host key identifies your SSH server to clients. Keep it consistent across restarts to avoid "host key changed" warnings
  - SSH keys are gitignored (see `.gitignore`)
- `DATABASE_PATH` - Path to SQLite database file (default: `data/terminal.db`)
- `JWT_SECRET` - Secret key for JWT tokens (default: `change-this-secret-key-in-production`)

### Web Server

- `WEB_HOST` - Host to bind to (default: same as `HOST` or `0.0.0.0`)
- `WEB_PORT` - Port to listen on (default: `8080`)
- `DATABASE_PATH` - Path to SQLite database file (default: `data/terminal.db`, used when `DATABASE_URL` is not set)
- `DATABASE_URL` - PostgreSQL connection URL (optional, if set, uses PostgreSQL instead of SQLite)
- `JWT_SECRET` - Secret key for JWT tokens (default: `change-this-secret-key-in-production`)

### Shared Configuration

Both servers use the same database by default. For separate deployments, you can:

- Use the same `DATABASE_PATH` for SQLite (file must be accessible to both)
- Use `DATABASE_URL` for PostgreSQL (recommended for separate containers)

### Database Options

The server supports both **SQLite** (default) and **PostgreSQL**. Switching is automatic based on configuration:

#### SQLite (Default)

By default, the server uses SQLite. The database file is created at `data/terminal.db`. The `data/` directory will be created automatically if it doesn't exist.

**Configuration:**

- `DATABASE_PATH` - Path to SQLite database file (default: `data/terminal.db`)
- `DATABASE_URL` - Leave unset or empty

**Examples:**

```bash
# Using default (data/terminal.db)
./bin/terminal.sh

# Custom SQLite location
DATABASE_PATH=/var/lib/terminal.sh/terminal.db ./bin/terminal.sh

# In-memory SQLite (testing only)
DATABASE_PATH=:memory: ./bin/terminal.sh
```

#### PostgreSQL

To use PostgreSQL, set the `DATABASE_URL` environment variable. When `DATABASE_URL` is set, `DATABASE_PATH` is ignored.

**Configuration:**

- `DATABASE_URL` - PostgreSQL connection URL (required)
- Format: `postgres://user:password@host:port/dbname` or `postgresql://user:password@host:port/dbname`

**Examples:**

```bash
# Local PostgreSQL
DATABASE_URL=postgres://user:password@localhost:5432/terminal_sh ./bin/terminal.sh

# Remote PostgreSQL
DATABASE_URL=postgres://user:password@db.example.com:5432/terminal_sh ./bin/terminal.sh

# PostgreSQL with SSL
DATABASE_URL=postgres://user:password@db.example.com:5432/terminal_sh?sslmode=require ./bin/terminal.sh
```

**Note:** The `data/` directory and all `.db` files are gitignored, so your SQLite database won't be committed to the repository.

## Project Structure

```
terminal.sh/
├── cmd/
│   ├── all/
│   │   └── main.go          # Combined server entry point (SSH + Web)
│   ├── ssh/
│   │   └── main.go          # SSH server entry point
│   └── web/
│       └── main.go          # WebSocket server entry point
├── terminal/
│   ├── ssh/                 # SSH-specific terminal code
│   │   └── server.go
│   ├── websocket/           # WebSocket-specific terminal code
│   │   ├── server.go
│   │   ├── bridge.go
│   │   ├── messages.go
│   │   └── http.go
│   ├── login.go             # Shared login model
│   ├── shell.go             # Shared shell model
│   ├── chat.go              # Chat UI model
│   └── ...
├── cmd/                     # Command handlers
│   ├── chat_commands.go     # Chat command handlers
│   └── ...
├── services/                # Business logic
│   ├── chat.go              # Chat service
│   └── ...
├── models/                  # Data models
│   ├── chat.go              # Chat data models
│   └── ...
├── database/               # Database layer
├── config/                  # Configuration
├── filesystem/              # Virtual filesystem
├── web/                     # Frontend files (HTML, JS, CSS)
│   ├── index.html
│   ├── terminal.js
│   └── style.css
└── Dockerfile               # Docker build file
```

## Docker Deployment

### Build Images

**Combined Server (Default):**

```bash
docker build -t terminal.sh .
```

**SSH Server Only:**

```bash
docker build -t terminal.sh-ssh --target ssh .
```

**Web Server Only:**

```bash
docker build -t terminal.sh-web --target web .
```

### Run Containers

**Combined Server (Recommended):**

```bash
docker run -p 2222:2222 -p 8080:8080 \
  -e PORT=2222 \
  -e WEB_PORT=8080 \
  -e DATABASE_PATH=/data/terminal.db \
  -v $(pwd)/data:/data \
  terminal.sh
```

**Combined Server - with .env file:**

```bash
docker run -p 2222:2222 -p 8080:8080 \
  --env-file .env \
  -v $(pwd)/data:/data \
  terminal.sh
```

**SSH Server Only:**

```bash
docker run -p 2222:2222 \
  -e PORT=2222 \
  -v $(pwd)/data:/data \
  terminal.sh-ssh
```

**SSH Server Only - with .env file:**

```bash
docker run -p 2222:2222 \
  --env-file .env \
  -v $(pwd)/data:/data \
  terminal.sh-ssh
```

**Web Server Only:**

```bash
docker run -p 8080:8080 \
  -e WEB_PORT=8080 \
  -v $(pwd)/data:/data \
  terminal.sh-web
```

**Web Server Only - with .env file:**

```bash
docker run -p 8080:8080 \
  --env-file .env \
  -v $(pwd)/data:/data \
  terminal.sh-web
```

**Note:** The default `DATABASE_PATH` is `data/terminal.db`, which will be created in the container at `/data/terminal.db` when you mount `$(pwd)/data:/data`.

### Docker Compose (Both Servers)

#### With SQLite (Shared File)

For deploying both servers with a shared SQLite database:

```yaml
version: "3.8"

services:
  ssh-server:
    build:
      context: .
      target: ssh
    env_file:
      - .env # Optional: load from .env file
    environment:
      - PORT=2222
      - DATABASE_PATH=/data/terminal.db
    volumes:
      - ./data:/data
    ports:
      - "2222:2222"

  web-server:
    build:
      context: .
      target: web
    env_file:
      - .env # Optional: load from .env file
    environment:
      - WEB_PORT=8080
      - DATABASE_PATH=/data/terminal.db
    volumes:
      - ./data:/data
    ports:
      - "8080:8080"
```

#### With PostgreSQL (Recommended for Production)

For deploying both servers with PostgreSQL:

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:15
    environment:
      - POSTGRES_USER=terminal_sh
      - POSTGRES_PASSWORD=changeme
      - POSTGRES_DB=terminal_sh
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  ssh-server:
    build:
      context: .
      target: ssh
    env_file:
      - .env # Optional: load from .env file
    environment:
      - PORT=2222
      - DATABASE_URL=postgres://terminal_sh:changeme@postgres:5432/terminal_sh
    depends_on:
      - postgres
    ports:
      - "2222:2222"

  web-server:
    build:
      context: .
      target: web
    env_file:
      - .env # Optional: load from .env file
    environment:
      - WEB_PORT=8080
      - DATABASE_URL=postgres://terminal_sh:changeme@postgres:5432/terminal_sh
    depends_on:
      - postgres
    ports:
      - "8080:8080"

volumes:
  postgres_data:
```

**Note:**

- SQLite: The `data/` directory will be created automatically on first run. Both servers share the same database file.
- PostgreSQL: Both servers connect to the same PostgreSQL database. Recommended for production and multi-container deployments.

## Development

### Prerequisites

- Go 1.25.5 or later
- SQLite (for default database)

### Running in Development

**Option 1: Combined (Recommended)**

```bash
go run ./cmd/all
```

This starts both servers. Connect via:

- SSH: `ssh -p 2222 username@localhost`
- Web: Open `http://localhost:8080` in your browser

**Option 2: Separate**

1. **Start SSH server:**

   ```bash
   go run ./cmd/ssh
   ```

2. **Start Web server (in another terminal):**

   ```bash
   go run ./cmd/web
   ```

3. **Connect:**
   - SSH: `ssh -p 2222 username@localhost`
   - Web: Open `http://localhost:8080` in your browser

### Testing

Both servers provide identical functionality. For gameplay testing, see [GAMEPLAY.md](GAMEPLAY.md).

## License

ISC

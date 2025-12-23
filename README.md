# SSH4XX Server

A simple SSH server with a terminal interface built in Go, featuring a virtual filesystem and basic command support.

## Features

- Real SSH server implementation using `golang.org/x/crypto/ssh`
- Beautiful terminal interface with Charm libraries (lipgloss)
- Animated welcome banner and help command
- Virtual filesystem with basic navigation
- Core commands: `pwd`, `ls`, `cd`, `cat`, `clear`, `help`

## Building

```bash
go build -o ssh4xx-server .
```

## Running

```bash
./ssh4xx-server
```

The server will listen on `0.0.0.0:2222` by default.

## Configuration

Environment variables:

- `HOST` - Host to bind to (default: `0.0.0.0`)
- `PORT` - Port to listen on (default: `2222`)
- `HOSTKEY_PATH` - Path to SSH host key file (optional, generates new key if not provided)

Example:

```bash
PORT=2222 HOST=0.0.0.0 ./ssh4xx-server
```

## Connecting

Connect using SSH client:

```bash
ssh -p 2222 <username>@your-server-ip
```

**Authentication:**

- The server uses password authentication
- **Auto-registration**: Any username/password combination will automatically create a new account on first login
- After registration, use the same credentials to log in
- Example: `ssh -p 2222 daniel@localhost` (password can be anything on first login)

## Docker Deployment

Build the Docker image:

```bash
docker build -t ssh4xx-server .
```

Run the container:

```bash
docker run -p 2222:2222 -e PORT=2222 ssh4xx-server
```

## Coolify Deployment

1. Push this repository to your git provider
2. In Coolify, create a new application
3. Select "Docker" as the build type
4. Set the port to `2222`
5. Optionally set environment variables for configuration
6. Deploy!

## Development

Project structure:

```
ssh4xx-go/
├── main.go              # Entry point
├── cmd/                 # Command handlers
├── config/              # Configuration management
├── filesystem/          # Virtual filesystem
├── terminal/            # Terminal interface & SSH server
└── Dockerfile           # Docker build file
```

## License

ISC

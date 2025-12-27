# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build SSH server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o terminal.sh-ssh ./cmd/ssh

# Build Web server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o terminal.sh-web ./cmd/web

# Build Combined server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o terminal.sh ./cmd/all

# Combined Server stage (default)
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the combined binary from builder
COPY --from=builder /app/terminal.sh .

# Copy web directory for static files
COPY --from=builder /app/web ./web

# Expose both ports
EXPOSE 2222 8080

# Run the combined server
CMD ["./terminal.sh"]

# SSH Server stage
FROM alpine:latest AS ssh

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the SSH binary from builder
COPY --from=builder /app/terminal.sh-ssh .

# Copy web directory for static files (not needed for SSH, but kept for consistency)
COPY --from=builder /app/web ./web

# Expose SSH port (default 2222)
EXPOSE 2222

# Run the SSH server
CMD ["./terminal.sh-ssh"]

# Web Server stage
FROM alpine:latest AS web

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Web binary from builder
COPY --from=builder /app/terminal.sh-web .

# Copy web directory for static files
COPY --from=builder /app/web ./web

# Expose Web port (default 8080)
EXPOSE 8080

# Run the Web server
CMD ["./terminal.sh-web"]

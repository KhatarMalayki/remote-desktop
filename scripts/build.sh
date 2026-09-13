#!/bin/bash
set -e

echo "Building RemoteDesk..."

VERSION=${VERSION:-"0.1.0"}

# Build server (needs CGO for SQLite)
echo "=> Building server..."
CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$VERSION" -o bin/rd-server ./cmd/server

# Build agent for multiple platforms (no CGO needed)
echo "=> Building agent (linux/amd64)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/rd-agent-linux-amd64 ./cmd/agent

echo "=> Building agent (linux/arm64)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/rd-agent-linux-arm64 ./cmd/agent

echo "=> Building agent (windows/amd64)..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/rd-agent-windows-amd64.exe ./cmd/agent

echo "=> Building agent (darwin/amd64)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/rd-agent-darwin-amd64 ./cmd/agent

echo "=> Building agent (darwin/arm64)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/rd-agent-darwin-arm64 ./cmd/agent

echo ""
echo "Build complete! Binaries in ./bin/"
ls -lh bin/

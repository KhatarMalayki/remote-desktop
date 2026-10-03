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

if [ "$(uname -s)" = Darwin ]; then
  for arch in amd64 arm64; do
    echo "=> Building native agent (darwin/$arch)..."
    CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS='-mmacosx-version-min=13.0' CGO_LDFLAGS='-mmacosx-version-min=13.0' GOOS=darwin GOARCH="$arch" go build -ldflags="-s -w" -o "bin/rd-agent-darwin-$arch" ./cmd/agent
  done
else
  echo 'macOS agents skipped: native capture/input require the macOS CI builder (CGO enabled).'
fi

echo ""
echo "Build complete! Binaries in ./bin/"
ls -lh bin/

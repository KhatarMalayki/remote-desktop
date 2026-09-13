#!/bin/bash
set -e

echo "==> Pulling latest code..."
cd "$(dirname "$0")/.."
git pull

echo "==> Building rd-server..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o /usr/local/bin/rd-server ./cmd/server

echo "==> Restarting remotedesk service..."
sudo systemctl restart remotedesk

echo "==> Done! Status: $(sudo systemctl is-active remotedesk)"

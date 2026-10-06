# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

RUN CGO_ENABLED=0 GOBIN=/wireguard go install golang.zx2c4.com/wireguard@v0.0.0-20231211153847-12269c276173

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY prebuilt-agent/ /prebuilt-agent/

# Build server
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /bin/rd-server ./cmd/server

# Build agent binaries for multi-platform distribution & client auto-update
RUN mkdir -p /agents && \
    if [ -f /prebuilt-agent/rd-agent-windows-amd64.exe ] && [ -f /prebuilt-agent/rd-agent-uiaccess.exe ]; then cp /prebuilt-agent/rd-agent-windows-amd64.exe /agents/rd-agent-windows-amd64.exe && cp /prebuilt-agent/rd-agent-uiaccess.exe /agents/rd-agent-uiaccess.exe; else echo "ERROR: signed Windows agent artifacts missing" >&2; exit 1; fi && \
    if [ -f /prebuilt-agent/wireguard.exe ]; then cp /prebuilt-agent/wireguard.exe /agents/wireguard.exe; fi && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /agents/rd-agent-linux-amd64 ./cmd/agent && \
    test -s /prebuilt-agent/rd-agent-darwin-amd64 && test -s /prebuilt-agent/rd-agent-darwin-arm64 && \
    cp /prebuilt-agent/rd-agent-darwin-amd64 /prebuilt-agent/rd-agent-darwin-arm64 /agents/ && \
    chmod 755 /agents/rd-agent-darwin-*

# Runtime stage - minimal & secure
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata iproute2 iptables iptables-legacy wireguard-tools && \
    ln -sf /sbin/iptables-legacy /sbin/iptables && \
    iptables --version | grep -q '(legacy)'

COPY --from=builder /bin/rd-server /usr/local/bin/rd-server
COPY --from=builder /wireguard/wireguard /usr/local/bin/wireguard-go
COPY --from=builder /agents /app/agents
COPY certs/RemoteDesk-Internal-Root.cer /app/certs/RemoteDesk-Internal-Root.cer

RUN mkdir -p /data

EXPOSE 8080

ENV RD_ADMIN_USER=admin
ENV RD_ADMIN_PASS=changeme
ENV RD_API_KEY=
ENV RD_AGENTS_DIR=/app/agents

CMD ["rd-server", "-addr", ":8080", "-db", "/data/devices.db", "-agents-dir", "/app/agents"]

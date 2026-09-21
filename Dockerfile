# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY prebuilt-agent/ /prebuilt-agent/

# Build server
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /bin/rd-server ./cmd/server

# Build agent binaries for multi-platform distribution & client auto-update
RUN mkdir -p /agents && \
    if [ -f /prebuilt-agent/rd-agent-windows-amd64.exe ]; then cp /prebuilt-agent/rd-agent-windows-amd64.exe /agents/rd-agent-windows-amd64.exe; else echo "ERROR: signed Windows agent artifact missing" >&2; exit 1; fi && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /agents/rd-agent-linux-amd64 ./cmd/agent && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o /agents/rd-agent-darwin-amd64 ./cmd/agent

# Runtime stage - minimal & secure
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata

COPY --from=builder /bin/rd-server /usr/local/bin/rd-server
COPY --from=builder /agents /app/agents
COPY certs/RemoteDesk-Internal-Root.cer /app/certs/RemoteDesk-Internal-Root.cer

RUN mkdir -p /data

EXPOSE 8080

ENV RD_ADMIN_USER=admin
ENV RD_ADMIN_PASS=changeme
ENV RD_API_KEY=
ENV RD_AGENTS_DIR=/app/agents

CMD ["rd-server", "-addr", ":8080", "-db", "/data/devices.db", "-agents-dir", "/app/agents"]

# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build server
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /bin/rd-server ./cmd/server

# Build agent binaries for multi-platform distribution & client auto-update
RUN mkdir -p /agents && \
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o /agents/rd-agent-windows-amd64.exe ./cmd/agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /agents/rd-agent-linux-amd64 ./cmd/agent && \
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o /agents/rd-agent-darwin-amd64 ./cmd/agent

# Runtime stage - minimal & secure
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata

COPY --from=builder /bin/rd-server /usr/local/bin/rd-server
COPY --from=builder /agents /app/agents

RUN mkdir -p /data

EXPOSE 8080

ENV RD_ADMIN_USER=admin
ENV RD_ADMIN_PASS=changeme
ENV RD_API_KEY=
ENV RD_AGENTS_DIR=/app/agents

CMD ["rd-server", "-addr", ":8080", "-db", "/data/devices.db", "-agents-dir", "/app/agents"]

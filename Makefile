.PHONY: build server agent clean run docker

build: server agent

server:
	CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/rd-server ./cmd/server

agent:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/rd-agent ./cmd/agent

run: server
	./bin/rd-server -addr :8080 -admin-pass admin123

clean:
	rm -rf bin/

docker:
	docker build -t remotedesk .

docker-run:
	docker run -d --name remotedesk -p 8080:8080 -v rd-data:/data \
		-e RD_ADMIN_PASS=changeme -e RD_API_KEY=my-secret-key \
		remotedesk

agent-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/rd-agent-linux-amd64 ./cmd/agent

agent-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/rd-agent-windows.exe ./cmd/agent

agent-all:
	bash scripts/build.sh

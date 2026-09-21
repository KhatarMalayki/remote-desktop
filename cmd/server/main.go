package main

import (
	"flag"
	"log"
	"os"

	remotedesktop "github.com/user/remote-desktop"
	"github.com/user/remote-desktop/internal/server"
)

var version = "0.2.22"

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "devices.db", "sqlite database path")
	apiKey := flag.String("api-key", envOr("RD_API_KEY", ""), "API key for agent auth")
	adminUser := flag.String("admin-user", envOr("RD_ADMIN_USER", "admin"), "admin username")
	adminPass := flag.String("admin-pass", envOr("RD_ADMIN_PASS", "admin123"), "admin password")
	agentsDir := flag.String("agents-dir", envOr("RD_AGENTS_DIR", "bin/agents"), "directory with agent binaries")
	flag.Parse()

	cfg := server.Config{
		Addr:      *addr,
		DBPath:    *dbPath,
		APIKey:    *apiKey,
		AdminUser: *adminUser,
		AdminPass: *adminPass,
		Version:   version,
		AgentsDir: *agentsDir,
	}

	srv, err := server.New(cfg, remotedesktop.WebFS)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	log.Printf("[server] version %s starting...", version)
	log.Fatal(srv.ListenAndServe())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

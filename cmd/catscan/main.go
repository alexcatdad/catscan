// Package main provides the CatScan entry point.
// CatScan is a local-only background dashboard that monitors GitHub repos.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/alexcatdad/catscan/internal/config"
	"github.com/alexcatdad/catscan/internal/server"
)

var (
	testMode = flag.Bool("test", false, "Enable test mode (use fixture data)")
)

func main() {
	flag.Parse()

	// Check for test mode
	if *testMode || os.Getenv("CATSCAN_TEST") == "1" {
		if err := runTestMode(); err != nil {
			log.Fatalf("Test mode failed: %v", err)
		}
		return
	}

	// Normal mode
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	srv, err := server.NewServer(&cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// runTestMode starts the server in test mode with fixture data.
func runTestMode() error {
	// Create test config
	cfg := config.Config{
		ScanPath:             getFixturePath("repos"),
		GitHubOwner:          "alexcatdad",
		Port:                 9527,
		LocalIntervalSeconds: 10,
		GitHubIntervalSeconds: 60,
		StaleDays:            90,
		AbandonedDays:        365,
		Notifications: config.NotificationConfig{
			ActionsChanged: true,
			NewRelease:     true,
			PROpened:       true,
		},
	}

	// Override port from environment if specified
	if port := os.Getenv("CATSCAN_TEST_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}

	srv, err := server.NewServer(&cfg)
	if err != nil {
		return fmt.Errorf("failed to create test server: %w", err)
	}

	log.Printf("Starting CatScan in test mode on port %d", cfg.Port)
	return srv.Start()
}

// getFixturePath returns the path to a fixture file or directory.
func getFixturePath(name string) string {
	// Check if we're running from the test directory
	if _, err := os.Stat("test/fixtures/" + name); err == nil {
		return "test/fixtures/" + name
	}
	// Try relative to repo root
	if _, err := os.Stat("../test/fixtures/" + name); err == nil {
		return "../test/fixtures/" + name
	}
	// Try absolute path from working directory
	wd, _ := os.Getwd()
	if _, err := os.Stat(wd + "/test/fixtures/" + name); err == nil {
		return wd + "/test/fixtures/" + name
	}
	// Fallback
	return "test/fixtures/" + name
}

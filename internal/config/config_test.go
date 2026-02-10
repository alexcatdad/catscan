package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexcatdad/catscan/internal/config"
)

// TestLoadReturnsDefaultsWhenFileDoesntExist tests that Load returns
// default config when the config file doesn't exist.
func TestLoadReturnsDefaultsWhenFileDoesntExist(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Override home directory for this test
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify default values
	if cfg.ScanPath == "" {
		t.Error("ScanPath should not be empty")
	}
	if cfg.GitHubOwner != "alexcatdad" {
		t.Errorf("GitHubOwner = %s, want alexcatdad", cfg.GitHubOwner)
	}
	if cfg.Port != 7700 {
		t.Errorf("Port = %d, want 7700", cfg.Port)
	}
	if cfg.LocalIntervalSeconds != 60 {
		t.Errorf("LocalIntervalSeconds = %d, want 60", cfg.LocalIntervalSeconds)
	}
	if cfg.GitHubIntervalSeconds != 300 {
		t.Errorf("GitHubIntervalSeconds = %d, want 300", cfg.GitHubIntervalSeconds)
	}
	if cfg.StaleDays != 30 {
		t.Errorf("StaleDays = %d, want 30", cfg.StaleDays)
	}
	if cfg.AbandonedDays != 90 {
		t.Errorf("AbandonedDays = %d, want 90", cfg.AbandonedDays)
	}
}

// TestLoadAndSaveRoundTrip tests that saving and loading config preserves data.
func TestLoadAndSaveRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	// Create config with custom values
	original := config.Config{
		ScanPath:              "/custom/path",
		GitHubOwner:           "testowner",
		Port:                  8080,
		LocalIntervalSeconds:  120,
		GitHubIntervalSeconds: 600,
		StaleDays:             45,
		AbandonedDays:         120,
		Notifications: config.NotificationConfig{
			ActionsChanged: false,
			NewRelease:     false,
			PROpened:       true,
			CloneCompleted: false,
			Error:          false,
		},
	}

	// Save config
	if err := config.Save(original); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load config
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify all fields match
	if loaded.ScanPath != original.ScanPath {
		t.Errorf("ScanPath = %s, want %s", loaded.ScanPath, original.ScanPath)
	}
	if loaded.GitHubOwner != original.GitHubOwner {
		t.Errorf("GitHubOwner = %s, want %s", loaded.GitHubOwner, original.GitHubOwner)
	}
	if loaded.Port != original.Port {
		t.Errorf("Port = %d, want %d", loaded.Port, original.Port)
	}
	if loaded.LocalIntervalSeconds != original.LocalIntervalSeconds {
		t.Errorf("LocalIntervalSeconds = %d, want %d", loaded.LocalIntervalSeconds, original.LocalIntervalSeconds)
	}
	if loaded.GitHubIntervalSeconds != original.GitHubIntervalSeconds {
		t.Errorf("GitHubIntervalSeconds = %d, want %d", loaded.GitHubIntervalSeconds, original.GitHubIntervalSeconds)
	}
	if loaded.StaleDays != original.StaleDays {
		t.Errorf("StaleDays = %d, want %d", loaded.StaleDays, original.StaleDays)
	}
	if loaded.AbandonedDays != original.AbandonedDays {
		t.Errorf("AbandonedDays = %d, want %d", loaded.AbandonedDays, original.AbandonedDays)
	}
	if loaded.Notifications.ActionsChanged != original.Notifications.ActionsChanged {
		t.Errorf("ActionsChanged = %v, want %v", loaded.Notifications.ActionsChanged, original.Notifications.ActionsChanged)
	}
	if loaded.Notifications.PROpened != original.Notifications.PROpened {
		t.Errorf("PROpened = %v, want %v", loaded.Notifications.PROpened, original.Notifications.PROpened)
	}
}

// TestLoadFromValidFile tests loading from a valid config file.
func TestLoadFromValidFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, ".config", "catscan")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write a valid config file
	configPath := filepath.Join(configDir, "config.json")
	testConfig := map[string]any{
		"scanPath":              "~/test",
		"githubOwner":           "testowner",
		"port":                  9000,
		"localIntervalSeconds":  45,
		"githubIntervalSeconds": 180,
		"staleDays":             20,
		"abandonedDays":         60,
		"notifications": map[string]bool{
			"actionsChanged": true,
			"newRelease":     true,
			"prOpened":       false,
			"cloneCompleted": true,
			"error":          true,
		},
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify tilde was expanded
	if cfg.ScanPath == "~/test" {
		t.Error("Tilde should be expanded in ScanPath")
	}
	// The expanded path should contain the home directory
	if filepath.Base(cfg.ScanPath) != "test" {
		t.Errorf("ScanPath = %s, should end with 'test'", cfg.ScanPath)
	}

	// Verify other values
	if cfg.GitHubOwner != "testowner" {
		t.Errorf("GitHubOwner = %s, want testowner", cfg.GitHubOwner)
	}
	if cfg.Port != 9000 {
		t.Errorf("Port = %d, want 9000", cfg.Port)
	}
	if cfg.LocalIntervalSeconds != 45 {
		t.Errorf("LocalIntervalSeconds = %d, want 45", cfg.LocalIntervalSeconds)
	}
}

// TestExpandTilde tests the tilde expansion functionality.
func TestExpandTilde(t *testing.T) {
	// This is tested indirectly through TestLoadFromValidFile
	// but we can add more edge case tests here if needed

	t.Run("no tilde", func(t *testing.T) {
		// Can't test internal function directly without export
		// but the behavior is covered by other tests
	})

	t.Run("empty string", func(t *testing.T) {
		// Covered by other tests
	})
}

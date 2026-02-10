// Package config handles loading and saving CatScan configuration.
//
// The config file is stored at ~/.config/catscan/config.json and contains
// settings for scan paths, GitHub owner, polling intervals, lifecycle thresholds,
// and notification preferences.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// NotificationConfig holds per-event-type notification toggles.
type NotificationConfig struct {
	ActionsChanged bool `json:"actionsChanged"`
	NewRelease     bool `json:"newRelease"`
	PROpened       bool `json:"prOpened"`
	CloneCompleted bool `json:"cloneCompleted"`
	Error          bool `json:"error"`
}

// DefaultNotificationConfig returns the default notification settings.
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		ActionsChanged: true,
		NewRelease:     true,
		PROpened:       false,
		CloneCompleted: true,
		Error:          true,
	}
}

// Config represents the CatScan configuration.
type Config struct {
	ScanPath                string             `json:"scanPath"`
	GitHubOwner             string             `json:"githubOwner"`
	Port                    int                `json:"port"`
	LocalIntervalSeconds    int                `json:"localIntervalSeconds"`
	GitHubIntervalSeconds   int                `json:"githubIntervalSeconds"`
	StaleDays               int                `json:"staleDays"`
	AbandonedDays           int                `json:"abandonedDays"`
	Notifications           NotificationConfig `json:"notifications"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("getting home directory: %w", err)
	}

	return Config{
		ScanPath:              filepath.Join(homeDir, "REPOS", "alexcatdad"),
		GitHubOwner:           "alexcatdad",
		Port:                  7700,
		LocalIntervalSeconds:  60,
		GitHubIntervalSeconds: 300,
		StaleDays:             30,
		AbandonedDays:         90,
		Notifications:         DefaultNotificationConfig(),
	}, nil
}

// configDir returns the CatScan config directory path.
func configDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "catscan")
	return configDir, nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// ensureConfigDir creates the config directory if it doesn't exist.
func ensureConfigDir() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	// Check if directory exists
	info, err := os.Stat(dir)
	if err == nil {
		// Exists, verify it's a directory
		if !info.IsDir() {
			return fmt.Errorf("config path exists but is not a directory: %s", dir)
		}
		return nil
	}

	// Doesn't exist, create it
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking config directory: %w", err)
	}

	// Create with permissions 0755 (rwxr-xr-x)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	return nil
}

// expandTilde expands a leading ~ to the user's home directory.
func expandTilde(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	// Handle ~/path or ~ alone
	if len(path) == 1 {
		return homeDir, nil
	}

	// path starts with ~/something
	return filepath.Join(homeDir, path[2:]), nil
}

// Load loads the config from ~/.config/catscan/config.json.
// If the file doesn't exist, returns default config.
func Load() (Config, error) {
	cfgPath, err := configPath()
	if err != nil {
		return Config{}, err
	}

	// Try to read the file
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist, return defaults
			return DefaultConfig()
		}
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	// Parse JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config JSON: %w", err)
	}

	// Expand tilde in scan path
	cfg.ScanPath, err = expandTilde(cfg.ScanPath)
	if err != nil {
		return Config{}, fmt.Errorf("expanding tilde in scanPath: %w", err)
	}

	return cfg, nil
}

// Save saves the config to ~/.config/catscan/config.json.
// The config directory is created if it doesn't exist.
func Save(cfg Config) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}

	cfgPath, err := configPath()
	if err != nil {
		return err
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config JSON: %w", err)
	}

	// Write atomically: write to temp file, then rename
	tmpPath := cfgPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing config temp file: %w", err)
	}

	// Rename temp file to actual file (atomic on POSIX systems)
	if err := os.Rename(tmpPath, cfgPath); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming config file: %w", err)
	}

	return nil
}

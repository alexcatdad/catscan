// Package cache handles persistent storage of repository data and user state.
//
// cache.json stores the full list of Repo objects and is rebuilt on each poll cycle.
// state.json stores persistent user state like last-seen release tags.
// Both files are stored in ~/.config/catscan/ and written atomically.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexcatdad/catscan/internal/model"
)

// cacheDir returns the CatScan cache directory path (~/.config/catscan/).
func cacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".config", "catscan")
	return cacheDir, nil
}

// cachePath returns the full path to cache.json.
func cachePath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cache.json"), nil
}

// statePath returns the full path to state.json.
func statePath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// ensureCacheDir creates the cache directory if it doesn't exist.
func ensureCacheDir() error {
	dir, err := cacheDir()
	if err != nil {
		return err
	}

	// Check if directory exists
	info, err := os.Stat(dir)
	if err == nil {
		// Exists, verify it's a directory
		if !info.IsDir() {
			return fmt.Errorf("cache path exists but is not a directory: %s", dir)
		}
		return nil
	}

	// Doesn't exist, create it
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking cache directory: %w", err)
	}

	// Create with permissions 0755 (rwxr-xr-x)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	return nil
}

// writeAtomic writes data to a file atomically.
func writeAtomic(path string, data []byte) error {
	// Write to temp file first
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	// Rename temp file to actual file (atomic on POSIX systems)
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming file: %w", err)
	}

	return nil
}

// RepoState stores persistent user state per repository.
type RepoState map[string]*RepoStateEntry

// RepoStateEntry holds state data for a single repository.
type RepoStateEntry struct {
	LastSeenReleaseTag string `json:"lastSeenReleaseTag"`
}

// ReadRepos reads the full repo list from cache.json.
// If the file doesn't exist or is empty, returns an empty slice.
func ReadRepos() ([]model.Repo, error) {
	cachePath, err := cachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist, return empty list
			return []model.Repo{}, nil
		}
		return nil, fmt.Errorf("reading cache file: %w", err)
	}

	// Handle empty file
	if len(data) == 0 {
		return []model.Repo{}, nil
	}

	var repos []model.Repo
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, fmt.Errorf("parsing cache JSON: %w", err)
	}

	return repos, nil
}

// WriteRepos writes the full repo list to cache.json.
// The cache directory is created if it doesn't exist.
// Write is atomic (temp file + rename).
func WriteRepos(repos []model.Repo) error {
	if err := ensureCacheDir(); err != nil {
		return err
	}

	path, err := cachePath()
	if err != nil {
		return err
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache JSON: %w", err)
	}

	if err := writeAtomic(path, data); err != nil {
		return fmt.Errorf("writing cache atomically: %w", err)
	}

	return nil
}

// ReadState reads the persistent user state from state.json.
// If the file doesn't exist or is empty, returns an empty state map.
func ReadState() (RepoState, error) {
	statePath, err := statePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist, return empty state
			return RepoState{}, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	// Handle empty file
	if len(data) == 0 {
		return RepoState{}, nil
	}

	var state RepoState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state JSON: %w", err)
	}

	// Handle null map
	if state == nil {
		return RepoState{}, nil
	}

	return state, nil
}

// WriteState writes the persistent user state to state.json.
// The cache directory is created if it doesn't exist.
// Write is atomic (temp file + rename).
func WriteState(state RepoState) error {
	if err := ensureCacheDir(); err != nil {
		return err
	}

	path, err := statePath()
	if err != nil {
		return err
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state JSON: %w", err)
	}

	if err := writeAtomic(path, data); err != nil {
		return fmt.Errorf("writing state atomically: %w", err)
	}

	return nil
}

// Package scanner provides repository scanning functionality.
//
// The local subpackage handles local git repository discovery and state extraction,
// including branch detection, dirty state, and last commit date.
package scanner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	// gitBin is the absolute path to the git binary.
	// Using absolute path ensures the binary can be found even without PATH.
	gitBin = "/usr/bin/git"
)

// LocalRepo represents a locally discovered repository.
type LocalRepo struct {
	Name      string
	Path      string
	Branch    string
	Dirty     bool
	LastCommit time.Time
}

// DiscoverLocalRepos scans the given path for git repositories.
// Only scans one level deep (direct children of the scan path).
// Skips hidden directories (those starting with a dot).
// Returns a sorted list of discovered repositories.
func DiscoverLocalRepos(scanPath string) ([]string, error) {
	// Expand tilde if present
	if strings.HasPrefix(scanPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("expanding tilde: %w", err)
		}
		if len(scanPath) == 1 {
			scanPath = homeDir
		} else {
			scanPath = filepath.Join(homeDir, scanPath[2:])
		}
	}

	// Read directory entries
	entries, err := os.ReadDir(scanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Scan path doesn't exist, return empty list
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading scan path: %w", err)
	}

	var repos []string

	for _, entry := range entries {
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Skip non-directories
		if !entry.IsDir() {
			continue
		}

		// Check if it contains a .git folder
		gitPath := filepath.Join(scanPath, entry.Name(), ".git")
		info, err := os.Stat(gitPath)
		if err != nil {
			continue
		}

		// .git exists and is a directory
		if info.IsDir() {
			repos = append(repos, entry.Name())
		}
	}

	// Sort alphabetically (already sorted by ReadDir, but let's be explicit)
	// Note: Go's ReadDir already returns sorted entries, so this is a no-op
	// but we'll keep it for clarity and robustness

	return repos, nil
}

// GetGitState extracts the git state for a repository at the given path.
// Returns branch name, dirty status, and last commit date.
// Logs errors and returns zero values if git commands fail.
func GetGitState(repoPath string) (branch string, dirty bool, lastCommit time.Time, err error) {
	// Get current branch
	branch, err = runGitCommand(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", false, time.Time{}, fmt.Errorf("getting branch: %w", err)
	}

	// Get dirty status
	dirtyOutput, err := runGitCommand(repoPath, "status", "--porcelain")
	if err != nil {
		return "", false, time.Time{}, fmt.Errorf("getting dirty status: %w", err)
	}
	dirty = strings.TrimSpace(dirtyOutput) != ""

	// Get last commit date
	dateOutput, err := runGitCommand(repoPath, "log", "-1", "--format=%aI")
	if err != nil {
		return "", false, time.Time{}, fmt.Errorf("getting last commit: %w", err)
	}

	lastCommit, err = time.Parse(time.RFC3339, strings.TrimSpace(dateOutput))
	if err != nil {
		return "", false, time.Time{}, fmt.Errorf("parsing commit date: %w", err)
	}

	return branch, dirty, lastCommit, nil
}

// runGitCommand executes a git command in the given repository directory.
// Returns the command's stdout output.
func runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command(gitBin, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %v: %w (stderr: %s)", args, err, stderr.String())
	}

	return stdout.String(), nil
}

// FindClonedRepos builds a map of repo names to their local paths
// for repos that exist locally in the scan path.
func FindClonedRepos(repos []string, scanPath string) map[string]string {
	// Expand tilde if present
	if strings.HasPrefix(scanPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// If we can't expand, use original path
			// This shouldn't happen in practice
		} else {
			if len(scanPath) == 1 {
				scanPath = homeDir
			} else {
				scanPath = filepath.Join(homeDir, scanPath[2:])
			}
		}
	}

	cloned := make(map[string]string)

	for _, name := range repos {
		repoPath := filepath.Join(scanPath, name)
		gitPath := filepath.Join(repoPath, ".git")

		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			cloned[name] = repoPath
		}
	}

	return cloned
}

// CloneRepo clones a GitHub repository to the scan path.
// Returns a channel of status updates for progress tracking.
// Errors are sent through the channel as CloneError values.
func CloneRepo(owner, name, scanPath string) <-chan CloneStatus {
	statusChan := make(chan CloneStatus)

	go func() {
		defer close(statusChan)

		// Expand tilde if present
		if strings.HasPrefix(scanPath, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				statusChan <- CloneStatus{
					Repo:  name,
					State: CloneStateError,
					Error: fmt.Sprintf("expanding home directory: %v", err),
				}
				return
			}
			if len(scanPath) == 1 {
				scanPath = homeDir
			} else {
				scanPath = filepath.Join(homeDir, scanPath[2:])
			}
		}

		// Check if repo already exists
		repoPath := filepath.Join(scanPath, name)
		if _, err := os.Stat(repoPath); err == nil {
			statusChan <- CloneStatus{
				Repo:  name,
				State: CloneStateError,
				Error: "repository already exists",
			}
			return
		}

		// Send started status
		statusChan <- CloneStatus{
			Repo:  name,
			State: CloneStateStarted,
		}

		// Clone the repository
		url := fmt.Sprintf("https://github.com/%s/%s.git", owner, name)
		cmd := exec.Command(gitBin, "clone", url, repoPath)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			statusChan <- CloneStatus{
				Repo:  name,
				State: CloneStateError,
				Error: fmt.Sprintf("clone failed: %v (stderr: %s)", err, stderr.String()),
			}
			return
		}

		// Send completed status
		statusChan <- CloneStatus{
			Repo:  name,
			State: CloneStateCompleted,
		}
	}()

	return statusChan
}

// CloneState represents the state of a clone operation.
type CloneState string

const (
	CloneStateStarted   CloneState = "started"
	CloneStateCompleted CloneState = "completed"
	CloneStateError     CloneState = "error"
)

// CloneStatus represents a status update during a clone operation.
type CloneStatus struct {
	Repo  string
	State CloneState
	Error string
}

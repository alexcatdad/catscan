// Package scanner provides repository scanning functionality.
//
// The github subpackage handles GitHub data fetching via the gh CLI.
package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// ghBin is the absolute path to the gh binary.
	// We check multiple common installation paths.
	ghBinOptHomebrew = "/opt/homebrew/bin/gh"
	ghBinUsrLocal    = "/usr/local/bin/gh"
	ghBinUsr         = "/usr/bin/gh"
)

// ghNotFoundError is returned when gh CLI is not found.
type ghNotFoundError struct {
	msg string
}

func (e *ghNotFoundError) Error() string {
	return e.msg
}

// IsGHNotFound returns true if the error indicates gh CLI was not found.
func IsGHNotFound(err error) bool {
	_, ok := err.(*ghNotFoundError)
	return ok
}

// ghAuthError is returned when gh CLI is not authenticated.
type ghAuthError struct {
	msg string
}

func (e *ghAuthError) Error() string {
	return e.msg
}

// IsGHAuthError returns true if the error indicates gh authentication failure.
func IsGHAuthError(err error) bool {
	_, ok := err.(*ghAuthError)
	return ok
}

// findGH returns the path to the gh CLI binary, or an error if not found.
func findGH() (string, error) {
	paths := []string{ghBinOptHomebrew, ghBinUsrLocal, ghBinUsr}

	for _, path := range paths {
		if info, err := exec.LookPath("gh"); err == nil {
			return info, nil
		}
		// Also check the absolute path
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
	}

	return "", &ghNotFoundError{msg: "gh CLI not found at common paths: " + strings.Join(paths, ", ")}
}

// runGH executes a gh command and returns the stdout.
func runGH(args ...string) (string, error) {
	ghPath, err := findGH()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(ghPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		// Check for authentication failure
		if strings.Contains(errMsg, "not authenticated") || strings.Contains(errMsg, "GH_ENTERPRISE_TOKEN") || strings.Contains(errMsg, "GitHub Credentials") {
			return "", &ghAuthError{msg: "gh CLI not authenticated: " + errMsg}
		}
		return "", fmt.Errorf("gh %v: %w (stderr: %s)", args, err, errMsg)
	}

	return stdout.String(), nil
}

// GitHubRepo represents a GitHub repository from the gh CLI.
type GitHubRepo struct {
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Visibility      string           `json:"visibility"`
	HomepageURL     string           `json:"homepageUrl"`
	PrimaryLanguage *PrimaryLanguage `json:"primaryLanguage"`
	Topics          []string         `json:"repositoryTopics"`
	HasPages        bool             `json:"hasPages"`
	DefaultBranch   *DefaultBranch   `json:"defaultBranchRef"`
	LatestRelease   *LatestRelease   `json:"latestRelease"`

	// Per-repo data fetched separately (not from gh repo list JSON)
	OpenPRs       int           `json:"-"`
	ActionsStatus string        `json:"-"`
	FilePresence  *FilePresence `json:"-"`
}

// PrimaryLanguage represents the primary programming language.
type PrimaryLanguage struct {
	Name string `json:"name"`
}

// DefaultBranch represents the default branch reference.
type DefaultBranch struct {
	Name string `json:"name"`
}

// LatestRelease represents the latest release.
type LatestRelease struct {
	TagName     string `json:"tagName"`
	PublishedAt string `json:"publishedAt"`
}

// ListGitHubRepos lists all repositories for the given owner using gh CLI.
func ListGitHubRepos(owner string) ([]GitHubRepo, error) {
	output, err := runGH("repo", "list", owner, "--json", "name,description,visibility,homepageUrl,primaryLanguage,repositoryTopics,hasPages,defaultBranchRef,latestRelease", "--limit", "200")
	if err != nil {
		return nil, fmt.Errorf("listing repos: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return []GitHubRepo{}, nil
	}

	var repos []GitHubRepo
	if err := json.Unmarshal([]byte(output), &repos); err != nil {
		return nil, fmt.Errorf("parsing repo list JSON: %w", err)
	}

	return repos, nil
}

// GetPROpenCount returns the count of open pull requests for a repository.
func GetPROpenCount(owner, name string) (int, error) {
	output, err := runGH("pr", "list", "--repo", fmt.Sprintf("%s/%s", owner, name), "--state", "open", "--json", "number", "--limit", "100")
	if err != nil {
		return 0, fmt.Errorf("listing PRs: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return 0, nil
	}

	// Parse JSON array of PR objects
	var prs []struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return 0, fmt.Errorf("parsing PR list JSON: %w", err)
	}

	return len(prs), nil
}

// ActionsWorkflowRun represents a GitHub Actions workflow run.
type ActionsWorkflowRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// GetActionsStatus returns the latest Actions status for a repository.
func GetActionsStatus(owner, name string) (string, error) {
	output, err := runGH("run", "list", "--repo", fmt.Sprintf("%s/%s", owner, name), "--limit", "1", "--json", "status,conclusion")
	if err != nil {
		// If there are no workflows, gh returns an error
		if strings.Contains(err.Error(), "no runs found") || strings.Contains(err.Error(), "not found") {
			return "none", nil
		}
		return "none", fmt.Errorf("listing runs: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return "none", nil
	}

	var runs []ActionsWorkflowRun
	if err := json.Unmarshal([]byte(output), &runs); err != nil {
		return "none", fmt.Errorf("parsing runs JSON: %w", err)
	}

	if len(runs) == 0 {
		return "none", nil
	}

	// Map conclusion to status
	conclusion := runs[0].Conclusion
	switch conclusion {
	case "success":
		return "passing", nil
	case "failure":
		return "failing", nil
	default:
		// For other states (pending, skipped, etc.), check status
		status := runs[0].Status
		if status == "completed" && conclusion == "" {
			return "none", nil
		}
		return "none", nil
	}
}

// GetLatestRelease returns the latest release info for a repository.
// This is typically already available from the repo listing, but this
// function can be used for a refresh.
func GetLatestRelease(owner, name string) (*LatestRelease, error) {
	output, err := runGH("release", "view", "--repo", fmt.Sprintf("%s/%s", owner, name), "--json", "tagName,publishedAt")
	if err != nil {
		// No releases found
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no releases") {
			return nil, nil
		}
		return nil, fmt.Errorf("getting release: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	var release LatestRelease
	if err := json.Unmarshal([]byte(output), &release); err != nil {
		return nil, fmt.Errorf("parsing release JSON: %w", err)
	}

	return &release, nil
}

// GetBranchProtection checks if the default branch is protected.
func GetBranchProtection(owner, name, defaultBranch string) (bool, error) {
	_, err := runGH("api", fmt.Sprintf("repos/%s/%s/branches/%s/protection", owner, name, defaultBranch))
	if err != nil {
		// 404 means not protected
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		// 403 means insufficient permissions
		if strings.Contains(err.Error(), "403") {
			return false, nil
		}
		return false, fmt.Errorf("checking branch protection: %w", err)
	}

	// 200 means protected
	return true, nil
}

// FilePresence checks for the presence of specific files in a repository.
type FilePresence struct {
	HasREADME      bool
	HasLICENSE     bool
	HasCLAUDEmd    bool
	HasAGENTSmd    bool
	HasProjectJson bool
}

// GetFilePresence checks for the presence of specific files in a repository.
func GetFilePresence(owner, name string) (*FilePresence, error) {
	result := &FilePresence{}

	// Helper to check a file
	checkFile := func(path string) bool {
		_, err := runGH("api", fmt.Sprintf("repos/%s/%s/contents/%s", owner, name, path))
		return err == nil
	}

	// Check README and LICENSE (any README* or LICENSE* file)
	// We need to list the root directory to find these files
	rootOutput, err := runGH("api", fmt.Sprintf("repos/%s/%s/contents/", owner, name))
	if err == nil {
		var rootContents []struct {
			Name string `json:"name"`
		}
		if json.Unmarshal([]byte(rootOutput), &rootContents) == nil {
			for _, item := range rootContents {
				if !result.HasREADME && strings.HasPrefix(strings.ToUpper(item.Name), "README") {
					result.HasREADME = true
				}
				if !result.HasLICENSE && strings.HasPrefix(strings.ToUpper(item.Name), "LICENSE") {
					result.HasLICENSE = true
				}
			}
		}
	}

	// Check specific files
	result.HasCLAUDEmd = checkFile("CLAUDE.md")
	result.HasAGENTSmd = checkFile("AGENTS.md")
	result.HasProjectJson = checkFile(".project.json")

	return result, nil
}

// parseTime parses an RFC3339 timestamp.
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

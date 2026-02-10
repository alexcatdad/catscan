package scanner_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alexcatdad/catscan/internal/scanner"
)

// TestDiscoverLocalReposFindsRepos tests that DiscoverLocalRepos finds directories with .git folders.
func TestDiscoverLocalReposFindsRepos(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some directories with .git folders
	repo1Path := filepath.Join(tmpDir, "repo1")
	if err := os.MkdirAll(filepath.Join(repo1Path, ".git"), 0o755); err != nil {
		t.Fatalf("Failed to create repo1: %v", err)
	}

	repo2Path := filepath.Join(tmpDir, "repo2")
	if err := os.MkdirAll(filepath.Join(repo2Path, ".git"), 0o755); err != nil {
		t.Fatalf("Failed to create repo2: %v", err)
	}

	// Create a directory without .git
	nonRepoPath := filepath.Join(tmpDir, "non-repo")
	if err := os.MkdirAll(nonRepoPath, 0o755); err != nil {
		t.Fatalf("Failed to create non-repo: %v", err)
	}

	repos, err := scanner.DiscoverLocalRepos(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverLocalRepos() failed: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("len(repos) = %d, want 2", len(repos))
	}

	// Check that repo1 and repo2 are found
	foundRepo1 := false
	foundRepo2 := false
	for _, name := range repos {
		if name == "repo1" {
			foundRepo1 = true
		}
		if name == "repo2" {
			foundRepo2 = true
		}
	}

	if !foundRepo1 {
		t.Error("repo1 not found")
	}
	if !foundRepo2 {
		t.Error("repo2 not found")
	}
}

// TestDiscoverLocalReposSkipsNonGitDirectories tests that non-git directories are skipped.
func TestDiscoverLocalReposSkipsNonGitDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory without .git
	nonRepoPath := filepath.Join(tmpDir, "non-repo")
	if err := os.MkdirAll(nonRepoPath, 0o755); err != nil {
		t.Fatalf("Failed to create non-repo: %v", err)
	}

	repos, err := scanner.DiscoverLocalRepos(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverLocalRepos() failed: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("len(repos) = %d, want 0", len(repos))
	}
}

// TestDiscoverLocalReposSkipsHiddenDirectories tests that hidden directories are skipped.
func TestDiscoverLocalReposSkipsHiddenDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hidden directory with .git
	hiddenRepoPath := filepath.Join(tmpDir, ".hidden-repo")
	if err := os.MkdirAll(filepath.Join(hiddenRepoPath, ".git"), 0o755); err != nil {
		t.Fatalf("Failed to create hidden repo: %v", err)
	}

	// Create a visible directory with .git
	visibleRepoPath := filepath.Join(tmpDir, "visible-repo")
	if err := os.MkdirAll(filepath.Join(visibleRepoPath, ".git"), 0o755); err != nil {
		t.Fatalf("Failed to create visible repo: %v", err)
	}

	repos, err := scanner.DiscoverLocalRepos(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverLocalRepos() failed: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("len(repos) = %d, want 1", len(repos))
	}

	if len(repos) > 0 && repos[0] != "visible-repo" {
		t.Errorf("repos[0] = %s, want visible-repo", repos[0])
	}
}

// TestDiscoverLocalReposHandlesNonExistentPath tests that a non-existent path returns empty list.
func TestDiscoverLocalReposHandlesNonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "does-not-exist")

	repos, err := scanner.DiscoverLocalRepos(nonExistentPath)
	if err != nil {
		t.Fatalf("DiscoverLocalRepos() failed: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("len(repos) = %d, want 0", len(repos))
	}
}

// TestGetGitStateWithRealRepo tests git state extraction with a real temporary git repo.
func TestGetGitStateWithRealRepo(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a git repo
	initCmd := exec.Command("git", "init", repoPath)
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Configure git
	configCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configCmd.Dir = repoPath
	if err := configCmd.Run(); err != nil {
		t.Fatalf("Failed to config git: %v", err)
	}

	configCmd = exec.Command("git", "config", "user.name", "Test User")
	configCmd.Dir = repoPath
	if err := configCmd.Run(); err != nil {
		t.Fatalf("Failed to config git: %v", err)
	}

	// Create a commit
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	addCmd := exec.Command("git", "add", "test.txt")
	addCmd.Dir = repoPath
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "test commit")
	commitCmd.Dir = repoPath
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get git state
	branch, dirty, lastCommit, err := scanner.GetGitState(repoPath)
	if err != nil {
		t.Fatalf("GetGitState() failed: %v", err)
	}

	// Check results
	_ = branch // Used in the condition below
	if branch != "main" && branch != "master" {
		// Different git versions may have different default branches
		// Just verify it's not empty
		if branch == "" {
			t.Error("branch is empty, want main or master")
		}
	}

	if dirty {
		t.Error("dirty = true, want false (clean working tree)")
	}

	if lastCommit.IsZero() {
		t.Error("lastCommit is zero, want non-zero")
	}

	// Make the repo dirty
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Get git state again
	_, dirtyAgain, _, err := scanner.GetGitState(repoPath)
	if err != nil {
		t.Fatalf("GetGitState() failed on dirty repo: %v", err)
	}

	if !dirtyAgain {
		t.Error("dirty = false after modification, want true")
	}
}

// TestFindClonedRepos tests clone detection.
func TestFindClonedRepos(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a repo with .git
	repo1Path := filepath.Join(tmpDir, "repo1")
	if err := os.MkdirAll(filepath.Join(repo1Path, ".git"), 0o755); err != nil {
		t.Fatalf("Failed to create repo1: %v", err)
	}

	// Create a directory without .git
	repo2Path := filepath.Join(tmpDir, "repo2")
	if err := os.MkdirAll(repo2Path, 0o755); err != nil {
		t.Fatalf("Failed to create repo2: %v", err)
	}

	repos := []string{"repo1", "repo2", "repo3"}

	cloned := scanner.FindClonedRepos(repos, tmpDir)

	if len(cloned) != 1 {
		t.Errorf("len(cloned) = %d, want 1", len(cloned))
	}

	if path, ok := cloned["repo1"]; !ok {
		t.Error("repo1 not found in cloned map")
	} else if path != repo1Path {
		t.Errorf("repo1 path = %s, want %s", path, repo1Path)
	}

	if _, ok := cloned["repo2"]; ok {
		t.Error("repo2 should not be in cloned map (no .git)")
	}

	if _, ok := cloned["repo3"]; ok {
		t.Error("repo3 should not be in cloned map (doesn't exist)")
	}
}

// TestCloneRepoStarted tests that CloneRepo sends started status.
func TestCloneRepoStarted(t *testing.T) {
	// This test requires a real git clone to work
	// Skip in CI environments or when git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	t.Skip("clone test requires network access - skipping in unit tests")
}

// TestCloneRepoAlreadyExists tests that CloneRepo handles existing repos.
func TestCloneRepoAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an existing repo
	existingRepo := filepath.Join(tmpDir, "existing-repo")
	if err := os.MkdirAll(filepath.Join(existingRepo, ".git"), 0o755); err != nil {
		t.Fatalf("Failed to create existing repo: %v", err)
	}

	statusChan := scanner.CloneRepo("testowner", "existing-repo", tmpDir)

	// Receive status
	status := <-statusChan

	if status.State != scanner.CloneStateError {
		t.Errorf("state = %s, want %s", status.State, scanner.CloneStateError)
	}

	if status.Repo != "existing-repo" {
		t.Errorf("repo = %s, want existing-repo", status.Repo)
	}

	// Check error message contains "already exists"
	expectedMsg := "already exists"
	if status.Error == "" || !contains(status.Error, expectedMsg) {
		t.Errorf("error = %s, want to contain %s", status.Error, expectedMsg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

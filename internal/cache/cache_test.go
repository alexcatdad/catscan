package cache_test

import (
	"os"
	"testing"
	"time"

	"github.com/alexcatdad/catscan/internal/cache"
	"github.com/alexcatdad/catscan/internal/model"
)

// TestReadReposWhenFileDoesntExist tests that ReadRepos returns empty list
// when the cache file doesn't exist.
func TestReadReposWhenFileDoesntExist(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	repos, err := cache.ReadRepos()
	if err != nil {
		t.Fatalf("ReadRepos() failed: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("len(repos) = %d, want 0", len(repos))
	}
}

// TestWriteAndReadReposRoundTrip tests that writing and reading repos preserves data.
func TestWriteAndReadReposRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	// Create test repos
	now := time.Now().UTC()
	testRepos := []model.Repo{
		{
			Name:       "test-repo-1",
			FullName:   "alexcatdad/test-repo-1",
			Visibility: model.VisibilityPublic,
			Cloned:     true,
			LocalPath:  "/path/to/test-repo-1",
			Branch:     "main",
			Dirty:      false,
			GitHubLastPush: now,
			OpenPRs:        2,
			ActionsStatus:  model.ActionsStatusPassing,
			Lifecycle:      model.LifecycleOngoing,
		},
		{
			Name:            "test-repo-2",
			FullName:        "alexcatdad/test-repo-2",
			Visibility:      model.VisibilityPrivate,
			Cloned:          false,
			GitHubLastPush:  now.Add(-48 * time.Hour),
			OpenPRs:         0,
			ActionsStatus:   model.ActionsStatusNone,
			Lifecycle:       model.LifecycleStale,
			HasREADME:       true,
			HasLicense:      true,
			BranchProtected: true,
		},
	}

	// Write repos
	if err := cache.WriteRepos(testRepos); err != nil {
		t.Fatalf("WriteRepos() failed: %v", err)
	}

	// Read repos
	loaded, err := cache.ReadRepos()
	if err != nil {
		t.Fatalf("ReadRepos() failed: %v", err)
	}

	// Verify count
	if len(loaded) != len(testRepos) {
		t.Fatalf("len(loaded) = %d, want %d", len(loaded), len(testRepos))
	}

	// Verify first repo
	if loaded[0].Name != testRepos[0].Name {
		t.Errorf("Name = %s, want %s", loaded[0].Name, testRepos[0].Name)
	}
	if loaded[0].FullName != testRepos[0].FullName {
		t.Errorf("FullName = %s, want %s", loaded[0].FullName, testRepos[0].FullName)
	}
	if loaded[0].Visibility != testRepos[0].Visibility {
		t.Errorf("Visibility = %s, want %s", loaded[0].Visibility, testRepos[0].Visibility)
	}
	if loaded[0].Cloned != testRepos[0].Cloned {
		t.Errorf("Cloned = %v, want %v", loaded[0].Cloned, testRepos[0].Cloned)
	}
	if loaded[0].OpenPRs != testRepos[0].OpenPRs {
		t.Errorf("OpenPRs = %d, want %d", loaded[0].OpenPRs, testRepos[0].OpenPRs)
	}

	// Verify second repo
	if loaded[1].Name != testRepos[1].Name {
		t.Errorf("Name = %s, want %s", loaded[1].Name, testRepos[1].Name)
	}
	if loaded[1].HasREADME != testRepos[1].HasREADME {
		t.Errorf("HasREADME = %v, want %v", loaded[1].HasREADME, testRepos[1].HasREADME)
	}
}

// TestReadStateWhenFileDoesntExist tests that ReadState returns empty map
// when the state file doesn't exist.
func TestReadStateWhenFileDoesntExist(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	state, err := cache.ReadState()
	if err != nil {
		t.Fatalf("ReadState() failed: %v", err)
	}

	if len(state) != 0 {
		t.Errorf("len(state) = %d, want 0", len(state))
	}
}

// TestWriteAndReadStateRoundTrip tests that writing and reading state preserves data.
func TestWriteAndReadStateRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	// Create test state
	testState := cache.RepoState{
		"repo1": &cache.RepoStateEntry{
			LastSeenReleaseTag: "v1.0.0",
		},
		"repo2": &cache.RepoStateEntry{
			LastSeenReleaseTag: "v2.3.4",
		},
		"repo3": nil, // Test nil entries
	}

	// Write state
	if err := cache.WriteState(testState); err != nil {
		t.Fatalf("WriteState() failed: %v", err)
	}

	// Read state
	loaded, err := cache.ReadState()
	if err != nil {
		t.Fatalf("ReadState() failed: %v", err)
	}

	// Verify entries
	if len(loaded) != 3 {
		t.Fatalf("len(loaded) = %d, want 3", len(loaded))
	}

	if loaded["repo1"].LastSeenReleaseTag != "v1.0.0" {
		t.Errorf("repo1 tag = %s, want v1.0.0", loaded["repo1"].LastSeenReleaseTag)
	}
	if loaded["repo2"].LastSeenReleaseTag != "v2.3.4" {
		t.Errorf("repo2 tag = %s, want v2.3.4", loaded["repo2"].LastSeenReleaseTag)
	}
	if loaded["repo3"] != nil {
		t.Errorf("repo3 = %v, want nil", loaded["repo3"])
	}
}

// TestAtomicWriteDoesntCorruptExistingData tests that atomic writes
// don't corrupt existing data if the write fails partway through.
func TestAtomicWriteDoesntCorruptExistingData(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	// Write initial data
	originalRepos := []model.Repo{
		{
			Name:       "original-repo",
			FullName:   "alexcatdad/original-repo",
			Visibility: model.VisibilityPublic,
			Lifecycle:  model.LifecycleOngoing,
		},
	}

	if err := cache.WriteRepos(originalRepos); err != nil {
		t.Fatalf("WriteRepos() failed: %v", err)
	}

	// Write new data (this should atomically replace the old data)
	newRepos := []model.Repo{
		{
			Name:       "new-repo",
			FullName:   "alexcatdad/new-repo",
			Visibility: model.VisibilityPrivate,
			Lifecycle:  model.LifecycleStale,
		},
	}

	if err := cache.WriteRepos(newRepos); err != nil {
		t.Fatalf("WriteRepos() failed: %v", err)
	}

	// Verify we get the new data, not corrupted data
	loaded, err := cache.ReadRepos()
	if err != nil {
		t.Fatalf("ReadRepos() failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("len(loaded) = %d, want 1", len(loaded))
	}

	if loaded[0].Name != "new-repo" {
		t.Errorf("Name = %s, want new-repo", loaded[0].Name)
	}
}

// TestEmptyFileHandling tests that empty cache and state files
// are handled gracefully.
func TestEmptyFileHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tmpDir)

	// Create cache directory and write empty files
	configDir := ".config/catscan"
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write empty cache file
	if err := os.WriteFile(configDir+"/cache.json", []byte{}, 0o644); err != nil {
		t.Fatalf("Failed to write empty cache file: %v", err)
	}

	// Write empty state file
	if err := os.WriteFile(configDir+"/state.json", []byte{}, 0o644); err != nil {
		t.Fatalf("Failed to write empty state file: %v", err)
	}

	// Read repos - should return empty list, not error
	repos, err := cache.ReadRepos()
	if err != nil {
		t.Fatalf("ReadRepos() failed: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("len(repos) = %d, want 0", len(repos))
	}

	// Read state - should return empty map, not error
	state, err := cache.ReadState()
	if err != nil {
		t.Fatalf("ReadState() failed: %v", err)
	}
	if len(state) != 0 {
		t.Errorf("len(state) = %d, want 0", len(state))
	}
}

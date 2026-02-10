package scanner_test

import (
	"testing"
	"time"

	"github.com/alexcatdad/catscan/internal/cache"
	"github.com/alexcatdad/catscan/internal/model"
	"github.com/alexcatdad/catscan/internal/scanner"
)

// TestMergeGitHubOnlyRepo tests that a GitHub-only repo appears as not cloned.
func TestMergeGitHubOnlyRepo(t *testing.T) {
	localRepos := map[string]scanner.LocalRepo{}

	githubRepos := []scanner.GitHubRepo{
		{
			Name:        "test-repo",
			Description: "A test repo",
			Visibility:  "public",
			PrimaryLanguage: &scanner.PrimaryLanguage{
				Name: "Go",
			},
			DefaultBranch: &scanner.DefaultBranch{
				Name: "main",
			},
		},
	}

	state := cache.RepoState{}
	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	result := scanner.Merge(localRepos, githubRepos, "/test/path", state, thresholds)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	repo := result[0]

	if repo.Name != "test-repo" {
		t.Errorf("Name = %s, want test-repo", repo.Name)
	}

	if repo.Cloned {
		t.Error("Cloned = true, want false (GitHub-only repo)")
	}

	if repo.Description != "A test repo" {
		t.Errorf("Description = %s, want 'A test repo'", repo.Description)
	}

	if repo.Language != "Go" {
		t.Errorf("Language = %s, want Go", repo.Language)
	}

	if repo.Visibility != model.VisibilityPublic {
		t.Errorf("Visibility = %s, want public", repo.Visibility)
	}

	if repo.Branch != "main" {
		t.Errorf("Branch = %s, want main (from GitHub default branch)", repo.Branch)
	}
}

// TestMergeLocalOnlyRepo tests that a local-only repo appears with minimal data.
func TestMergeLocalOnlyRepo(t *testing.T) {
	now := time.Now().UTC()

	localRepos := map[string]scanner.LocalRepo{
		"local-repo": {
			Name:       "local-repo",
			Path:       "/path/to/local-repo",
			Branch:     "main",
			Dirty:      true,
			LastCommit: now,
		},
	}

	githubRepos := []scanner.GitHubRepo{}
	state := cache.RepoState{}
	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	result := scanner.Merge(localRepos, githubRepos, "/test/path", state, thresholds)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	repo := result[0]

	if repo.Name != "local-repo" {
		t.Errorf("Name = %s, want local-repo", repo.Name)
	}

	if !repo.Cloned {
		t.Error("Cloned = false, want true (local repo)")
	}

	if repo.LocalPath != "/path/to/local-repo" {
		t.Errorf("LocalPath = %s, want /path/to/local-repo", repo.LocalPath)
	}

	if !repo.Dirty {
		t.Error("Dirty = false, want true")
	}

	if !repo.LocalLastCommit.Equal(now) {
		t.Errorf("LocalLastCommit = %v, want %v", repo.LocalLastCommit, now)
	}

	// Minimal GitHub data should be empty/default
	if repo.Description != "" {
		t.Errorf("Description = %s, want empty (no GitHub data)", repo.Description)
	}
}

// TestMergeFullyMatchedRepo tests that a fully matched repo has all fields populated.
func TestMergeFullyMatchedRepo(t *testing.T) {
	now := time.Now().UTC()

	localRepos := map[string]scanner.LocalRepo{
		"matched-repo": {
			Name:       "matched-repo",
			Path:       "/path/to/matched-repo",
			Branch:     "main",
			Dirty:      false,
			LastCommit: now,
		},
	}

	githubRepos := []scanner.GitHubRepo{
		{
			Name:        "matched-repo",
			Description: "A matched repo",
			Visibility:  "public",
			HomepageURL: "https://example.com",
			PrimaryLanguage: &scanner.PrimaryLanguage{
				Name: "Go",
			},
			Topics: []scanner.RepositoryTopic{{Name: "tag1"}, {Name: "tag2"}},
			DefaultBranch: &scanner.DefaultBranch{
				Name: "main",
			},
			LatestRelease: &scanner.LatestRelease{
				TagName:     "v1.0.0",
				PublishedAt: "2024-01-01T00:00:00Z",
			},
		},
	}

	state := cache.RepoState{
		"matched-repo": &cache.RepoStateEntry{
			LastSeenReleaseTag: "v1.0.0",
		},
	}
	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	result := scanner.Merge(localRepos, githubRepos, "/test/path", state, thresholds)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	repo := result[0]

	// Local data
	if !repo.Cloned {
		t.Error("Cloned = false, want true")
	}
	if repo.Branch != "main" {
		t.Errorf("Branch = %s, want main", repo.Branch)
	}

	// GitHub data
	if repo.Description != "A matched repo" {
		t.Errorf("Description = %s, want 'A matched repo'", repo.Description)
	}
	if repo.HomepageURL != "https://example.com" {
		t.Errorf("HomepageURL = %s, want https://example.com", repo.HomepageURL)
	}
	if repo.Language != "Go" {
		t.Errorf("Language = %s, want Go", repo.Language)
	}
	if len(repo.Topics) != 2 {
		t.Errorf("len(Topics) = %d, want 2", len(repo.Topics))
	}
	if !repo.Completeness.HasHomepage {
		t.Error("Completeness.HasHomepage = false, want true")
	}

	// Release info
	if repo.LatestRelease == nil {
		t.Error("LatestRelease = nil, want non-nil")
	} else {
		if repo.LatestRelease.TagName != "v1.0.0" {
			t.Errorf("LatestRelease.TagName = %s, want v1.0.0", repo.LatestRelease.TagName)
		}
	}

	// Not a new release since we've seen it before
	if repo.NewRelease {
		t.Error("NewRelease = true, want false (already seen v1.0.0)")
	}
}

// TestMergeLifecycleComputed tests that lifecycle status is computed correctly.
func TestMergeLifecycleComputed(t *testing.T) {
	now := time.Now().UTC()

	localRepos := map[string]scanner.LocalRepo{
		"repo-with-prs": {
			Name:       "repo-with-prs",
			Path:       "/path/to/repo-with-prs",
			Branch:     "main",
			Dirty:      false,
			LastCommit: now.Add(-60 * 24 * time.Hour), // 60 days ago
		},
		"stale-repo": {
			Name:       "stale-repo",
			Path:       "/path/to/stale-repo",
			Branch:     "main",
			Dirty:      false,
			LastCommit: now.Add(-60 * 24 * time.Hour), // 60 days ago
		},
	}

	githubRepos := []scanner.GitHubRepo{
		{
			Name: "repo-with-prs",
			PrimaryLanguage: &scanner.PrimaryLanguage{
				Name: "Go",
			},
			DefaultBranch: &scanner.DefaultBranch{
				Name: "main",
			},
		},
		{
			Name: "stale-repo",
			PrimaryLanguage: &scanner.PrimaryLanguage{
				Name: "Go",
			},
			DefaultBranch: &scanner.DefaultBranch{
				Name: "main",
			},
		},
	}

	state := cache.RepoState{}
	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	result := scanner.Merge(localRepos, githubRepos, "/test/path", state, thresholds)

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// Find each repo
	var repoWithPRs, staleRepo *model.Repo
	for i := range result {
		if result[i].Name == "repo-with-prs" {
			repoWithPRs = &result[i]
		}
		if result[i].Name == "stale-repo" {
			staleRepo = &result[i]
		}
	}

	if repoWithPRs == nil {
		t.Fatal("repo-with-prs not found")
	}
	if staleRepo == nil {
		t.Fatal("stale-repo not found")
	}

	// Set PR count on repo-with-prs to make it ongoing
	repoWithPRs.OpenPRs = 1
	// Re-compute lifecycle with PRs
	repoWithPRs.Lifecycle = repoWithPRs.ComputeLifecycle(thresholds)

	// Repo with PRs should be ongoing
	if repoWithPRs.Lifecycle != model.LifecycleOngoing {
		t.Errorf("repo-with-prs Lifecycle = %s, want %s (has PRs)", repoWithPRs.Lifecycle, model.LifecycleOngoing)
	}

	// Stale repo has no activity
	// GitHubLastPush is zero since we didn't set it, no PRs, no CI
	if staleRepo.Lifecycle != model.LifecycleStale {
		t.Errorf("stale-repo Lifecycle = %s, want %s (OpenPRs=%d, ActionsStatus=%q, GitHubLastPush=%v)",
			staleRepo.Lifecycle, model.LifecycleStale, staleRepo.OpenPRs, staleRepo.ActionsStatus, staleRepo.GitHubLastPush)
	}
}

// TestMergeNewReleaseDetection tests that new releases are detected correctly.
func TestMergeNewReleaseDetection(t *testing.T) {
	localRepos := map[string]scanner.LocalRepo{}

	githubRepos := []scanner.GitHubRepo{
		{
			Name: "test-repo",
			PrimaryLanguage: &scanner.PrimaryLanguage{
				Name: "Go",
			},
			DefaultBranch: &scanner.DefaultBranch{
				Name: "main",
			},
			LatestRelease: &scanner.LatestRelease{
				TagName:     "v2.0.0",
				PublishedAt: "2024-01-01T00:00:00Z",
			},
		},
	}

	// State shows we've seen v1.0.0, so v2.0.0 is new
	state := cache.RepoState{
		"test-repo": &cache.RepoStateEntry{
			LastSeenReleaseTag: "v1.0.0",
		},
	}
	thresholds := model.LifecycleThresholds{}

	result := scanner.Merge(localRepos, githubRepos, "/test/path", state, thresholds)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if !result[0].NewRelease {
		t.Error("NewRelease = false, want true (v2.0.0 is new)")
	}
}

// TestMergeNoPreviousRelease tests that a release is new if no previous release seen.
func TestMergeNoPreviousRelease(t *testing.T) {
	localRepos := map[string]scanner.LocalRepo{}

	githubRepos := []scanner.GitHubRepo{
		{
			Name: "test-repo",
			PrimaryLanguage: &scanner.PrimaryLanguage{
				Name: "Go",
			},
			DefaultBranch: &scanner.DefaultBranch{
				Name: "main",
			},
			LatestRelease: &scanner.LatestRelease{
				TagName:     "v1.0.0",
				PublishedAt: "2024-01-01T00:00:00Z",
			},
		},
	}

	state := cache.RepoState{} // No entry for this repo
	thresholds := model.LifecycleThresholds{}

	result := scanner.Merge(localRepos, githubRepos, "/test/path", state, thresholds)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if !result[0].NewRelease {
		t.Error("NewRelease = false, want true (first release seen)")
	}
}

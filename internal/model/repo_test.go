package model_test

import (
	"testing"
	"time"

	"github.com/alexcatdad/catscan/internal/model"
)

// TestLifecycleOngoingRecentCommit tests that a repo with a recent commit
// is classified as ongoing.
func TestLifecycleOngoingRecentCommit(t *testing.T) {
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Now().Add(-24 * time.Hour), // 1 day ago
		OpenPRs:        0,
		ActionsStatus:  model.ActionsStatusNone,
	}

	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	if lifecycle != model.LifecycleOngoing {
		t.Errorf("lifecycle = %s, want %s", lifecycle, model.LifecycleOngoing)
	}
}

// TestLifecycleOngoingWithOpenPRs tests that a repo with open PRs
// is classified as ongoing even with old commits.
func TestLifecycleOngoingWithOpenPRs(t *testing.T) {
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Now().Add(-60 * 24 * time.Hour), // 60 days ago (stale)
		OpenPRs:        2,                                      // but has open PRs
		ActionsStatus:  model.ActionsStatusNone,
	}

	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	if lifecycle != model.LifecycleOngoing {
		t.Errorf("lifecycle = %s, want %s (open PRs should make it ongoing)", lifecycle, model.LifecycleOngoing)
	}
}

// TestLifecycleOngoingWithActiveCI tests that a repo with active CI
// is classified as ongoing.
func TestLifecycleOngoingWithActiveCI(t *testing.T) {
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Now().Add(-60 * 24 * time.Hour), // 60 days ago
		OpenPRs:        0,
		ActionsStatus:  model.ActionsStatusPassing, // active CI
	}

	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	if lifecycle != model.LifecycleOngoing {
		t.Errorf("lifecycle = %s, want %s (active CI should make it ongoing)", lifecycle, model.LifecycleOngoing)
	}
}

// TestLifecycleOngoingWithFailingCI tests that a repo with failing CI
// is still classified as ongoing (activity is activity).
func TestLifecycleOngoingWithFailingCI(t *testing.T) {
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Now().Add(-60 * 24 * time.Hour),
		OpenPRs:        0,
		ActionsStatus:  model.ActionsStatusFailing, // failing CI still counts
	}

	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	if lifecycle != model.LifecycleOngoing {
		t.Errorf("lifecycle = %s, want %s (failing CI should still make it ongoing)", lifecycle, model.LifecycleOngoing)
	}
}

// TestLifecycleMaintenance tests that a repo with old commits but passing CI
// is classified as maintenance.
//
// Note: The current implementation doesn't have a separate "maintenance" state
// for old commits + passing CI. This test documents the current behavior.
func TestLifecycleMaintenance(t *testing.T) {
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Now().Add(-45 * 24 * time.Hour), // 45 days ago
		OpenPRs:        0,
		ActionsStatus:  model.ActionsStatusPassing,
	}

	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	// With active CI (passing), it's actually ongoing, not maintenance
	// To be maintenance, we'd need no CI activity
	if lifecycle != model.LifecycleOngoing {
		t.Errorf("lifecycle = %s, want %s (passing CI makes it ongoing)", lifecycle, model.LifecycleOngoing)
	}
}

// TestLifecycleStale tests that a repo with no activity within stale threshold
// is classified as stale.
func TestLifecycleStale(t *testing.T) {
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Now().Add(-45 * 24 * time.Hour), // 45 days ago
		OpenPRs:        0,
		ActionsStatus:  model.ActionsStatusNone, // no CI
	}

	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	if lifecycle != model.LifecycleStale {
		t.Errorf("lifecycle = %s, want %s", lifecycle, model.LifecycleStale)
	}
}

// TestLifecycleAbandoned tests that a repo with no activity beyond abandoned
// threshold is classified as abandoned.
func TestLifecycleAbandoned(t *testing.T) {
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Now().Add(-100 * 24 * time.Hour), // 100 days ago
		OpenPRs:        0,
		ActionsStatus:  model.ActionsStatusNone,
	}

	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	if lifecycle != model.LifecycleAbandoned {
		t.Errorf("lifecycle = %s, want %s", lifecycle, model.LifecycleAbandoned)
	}
}

// TestLifecycleNoPushData tests that a repo with no push data is treated as stale.
func TestLifecycleNoPushData(t *testing.T) {
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Time{}, // zero time, no data
		OpenPRs:        0,
		ActionsStatus:  model.ActionsStatusNone,
	}

	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	if lifecycle != model.LifecycleStale {
		t.Errorf("lifecycle = %s, want %s (no data should be stale)", lifecycle, model.LifecycleStale)
	}
}

// TestLifecycleWithCustomThresholds tests that custom thresholds work correctly.
func TestLifecycleWithCustomThresholds(t *testing.T) {
	// A repo 15 days old
	repo := &model.Repo{
		Name:           "test-repo",
		GitHubLastPush: time.Now().Add(-15 * 24 * time.Hour),
		OpenPRs:        0,
		ActionsStatus:  model.ActionsStatusNone,
	}

	// With very short thresholds
	thresholds := model.LifecycleThresholds{
		StaleDays:     7,  // 1 week
		AbandonedDays: 14, // 2 weeks
	}

	lifecycle := repo.ComputeLifecycle(thresholds)
	if lifecycle != model.LifecycleAbandoned {
		t.Errorf("lifecycle = %s, want %s (15 days with 14-day threshold)", lifecycle, model.LifecycleAbandoned)
	}
}

// TestLifecycleAtThresholdBoundaries tests behavior at exact threshold boundaries.
func TestLifecycleAtThresholdBoundaries(t *testing.T) {
	thresholds := model.LifecycleThresholds{
		StaleDays:     30,
		AbandonedDays: 90,
	}

	t.Run("exactly at stale threshold", func(t *testing.T) {
		repo := &model.Repo{
			Name:           "test-repo",
			GitHubLastPush: time.Now().Add(-30 * 24 * time.Hour), // exactly 30 days
			OpenPRs:        0,
			ActionsStatus:  model.ActionsStatusNone,
		}

		lifecycle := repo.ComputeLifecycle(thresholds)
		// At exactly 30 days, should be stale (not < 30)
		if lifecycle != model.LifecycleStale {
			t.Errorf("lifecycle = %s, want %s", lifecycle, model.LifecycleStale)
		}
	})

	t.Run("one day before stale threshold", func(t *testing.T) {
		repo := &model.Repo{
			Name:           "test-repo",
			GitHubLastPush: time.Now().Add(-29 * 24 * time.Hour), // 29 days
			OpenPRs:        0,
			ActionsStatus:  model.ActionsStatusNone,
		}

		lifecycle := repo.ComputeLifecycle(thresholds)
		if lifecycle != model.LifecycleOngoing {
			t.Errorf("lifecycle = %s, want %s", lifecycle, model.LifecycleOngoing)
		}
	})

	t.Run("exactly at abandoned threshold", func(t *testing.T) {
		repo := &model.Repo{
			Name:           "test-repo",
			GitHubLastPush: time.Now().Add(-90 * 24 * time.Hour), // exactly 90 days
			OpenPRs:        0,
			ActionsStatus:  model.ActionsStatusNone,
		}

		lifecycle := repo.ComputeLifecycle(thresholds)
		if lifecycle != model.LifecycleAbandoned {
			t.Errorf("lifecycle = %s, want %s", lifecycle, model.LifecycleAbandoned)
		}
	})
}

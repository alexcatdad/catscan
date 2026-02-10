// Package model defines the core data structures for CatScan.
//
// The Repo type represents a unified view of a repository combining
// local git state and GitHub metadata. Lifecycle classification is
// computed from activity signals.
package model

import "time"

// Lifecycle represents the lifecycle status of a repository.
type Lifecycle string

const (
	// LifecycleOngoing indicates the repository is actively developed.
	// Commits within threshold OR open PRs OR active CI.
	LifecycleOngoing Lifecycle = "ongoing"

	// LifecycleMaintenance indicates no recent commits but CI is passing.
	// Stable and maintained.
	LifecycleMaintenance Lifecycle = "maintenance"

	// LifecycleStale indicates no commits beyond stale threshold, no CI activity.
	LifecycleStale Lifecycle = "stale"

	// LifecycleAbandoned indicates no commits beyond abandoned threshold, no CI.
	LifecycleAbandoned Lifecycle = "abandoned"
)

// ActionsStatus represents the CI/CD status from GitHub Actions.
type ActionsStatus string

const (
	ActionsStatusPassing ActionsStatus = "passing"
	ActionsStatusFailing ActionsStatus = "failing"
	ActionsStatusNone    ActionsStatus = "none"
)

// Visibility represents the repository visibility.
type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// CompletenessInfo tracks which docs/files exist in a repo.
type CompletenessInfo struct {
	HasDescription bool `json:"HasDescription"`
	HasReadme      bool `json:"HasReadme"`
	HasLicense     bool `json:"HasLicense"`
	HasTopics      bool `json:"HasTopics"`
	HasPages       bool `json:"HasPages"`
	HasHomepage    bool `json:"HasHomepage"`
	HasProjectJson bool `json:"HasProjectJson"`
	HasClaudeMd    bool `json:"HasClaudeMd"`
	HasAgentsMd    bool `json:"HasAgentsMd"`
}

// Repo represents a unified view of a repository combining local git state
// and GitHub metadata.
type Repo struct {
	// Identity
	Name       string     `json:"Name"`
	FullName   string     `json:"FullName"`
	Visibility Visibility `json:"Visibility"`

	// Clone state
	Cloned    bool   `json:"Cloned"`
	LocalPath string `json:"LocalPath,omitempty"`

	// Local git (cloned repos only)
	Branch          string    `json:"Branch,omitempty"`
	Dirty           bool      `json:"Dirty,omitempty"`
	LocalLastCommit time.Time `json:"LocalLastCommit,omitempty"`

	// GitHub metadata
	Description string   `json:"Description,omitempty"`
	HomepageURL string   `json:"HomepageURL,omitempty"`
	Language    string   `json:"Language,omitempty"`
	Topics      []string `json:"Topics,omitempty"`

	// Completeness (nested for frontend consumption)
	Completeness CompletenessInfo `json:"Completeness"`

	// Activity
	GitHubLastPush time.Time     `json:"GitHubLastPush"`
	OpenPRs        int           `json:"OpenPRs"`
	ActionsStatus  ActionsStatus `json:"ActionsStatus"`
	LatestRelease  *ReleaseInfo  `json:"LatestRelease,omitempty"`
	NewRelease     bool          `json:"NewRelease"`

	// Computed
	Lifecycle Lifecycle `json:"Lifecycle"`
}

// ReleaseInfo represents a GitHub release.
type ReleaseInfo struct {
	TagName     string    `json:"TagName"`
	PublishedAt time.Time `json:"PublishedAt"`
}

// LifecycleThresholds defines the day thresholds for lifecycle classification.
type LifecycleThresholds struct {
	StaleDays     int
	AbandonedDays int
}

// ComputeLifecycle calculates the lifecycle status based on activity signals.
func (r *Repo) ComputeLifecycle(thresholds LifecycleThresholds) Lifecycle {
	now := time.Now()

	// Check for ongoing indicators
	// 1. Recent commits within stale threshold
	if !r.GitHubLastPush.IsZero() {
		daysSincePush := int(now.Sub(r.GitHubLastPush).Hours() / 24)
		if daysSincePush < thresholds.StaleDays {
			return LifecycleOngoing
		}
	}

	// 2. Open PRs indicate ongoing work
	if r.OpenPRs > 0 {
		return LifecycleOngoing
	}

	// 3. Active CI (passing or failing) indicates ongoing work
	if r.ActionsStatus != "" && r.ActionsStatus != ActionsStatusNone {
		return LifecycleOngoing
	}

	// At this point, no ongoing indicators
	if !r.GitHubLastPush.IsZero() {
		daysSincePush := int(now.Sub(r.GitHubLastPush).Hours() / 24)

		if daysSincePush >= thresholds.StaleDays && daysSincePush < thresholds.AbandonedDays {
			return LifecycleStale
		}

		if daysSincePush >= thresholds.AbandonedDays {
			return LifecycleAbandoned
		}
	}

	// No push data at all - treat as stale
	return LifecycleStale
}

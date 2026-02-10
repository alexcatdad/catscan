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

// Repo represents a unified view of a repository combining local git state
// and GitHub metadata.
type Repo struct {
	// Identity
	Name      string    `json:"name"`
	FullName  string    `json:"fullName"`
	Visibility Visibility `json:"visibility"`

	// Clone state
	Cloned    bool   `json:"cloned"`
	LocalPath string `json:"localPath,omitempty"`

	// Local git (cloned repos only)
	Branch           string    `json:"branch,omitempty"`
	Dirty            bool      `json:"dirty,omitempty"`
	LocalLastCommit  time.Time `json:"localLastCommit,omitempty"`

	// GitHub metadata
	Description     string   `json:"description,omitempty"`
	HomepageURL     string   `json:"homepageUrl,omitempty"`
	Language        string   `json:"language,omitempty"`
	Topics          []string `json:"topics,omitempty"`
	HasPages        bool     `json:"hasPages"`
	HasREADME       bool     `json:"hasReadme"`
	HasLicense      bool     `json:"hasLicense"`
	HasCLAUDEmd     bool     `json:"hasClaudeMd"`
	HasAGENTSmd     bool     `json:"hasAgentsMd"`
	HasProjectJson  bool     `json:"hasProjectJson"`
	BranchProtected bool     `json:"branchProtected"`

	// Activity
	GitHubLastPush time.Time     `json:"githubLastPush"`
	OpenPRs        int            `json:"openPrs"`
	ActionsStatus  ActionsStatus  `json:"actionsStatus"`
	LatestRelease  *ReleaseInfo  `json:"latestRelease,omitempty"`
	NewRelease     bool           `json:"newRelease"`

	// Computed
	Lifecycle Lifecycle `json:"lifecycle"`
}

// ReleaseInfo represents a GitHub release.
type ReleaseInfo struct {
	TagName     string    `json:"tagName"`
	PublishedAt time.Time `json:"publishedAt"`
}

// LifecycleThresholds defines the day thresholds for lifecycle classification.
type LifecycleThresholds struct {
	StaleDays      int
	AbandonedDays  int
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
	// Check if maintenance (old commits but CI was passing at some point)
	// Since we check for "no CI activity" above, if we reach here with
	// ActionsStatus == None, we need to look at commit age
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

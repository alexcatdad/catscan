// TypeScript types matching the Go backend API responses.

// Lifecycle represents the lifecycle status of a repository.
export type Lifecycle = "ongoing" | "maintenance" | "stale" | "abandoned";

// ActionsStatus represents the CI/CD status from GitHub Actions.
export type ActionsStatus = "none" | "passing" | "failing";

// Visibility represents repository visibility.
export type Visibility = "public" | "private";

// ReleaseInfo represents the latest release from GitHub.
export interface ReleaseInfo {
	TagName: string;
	PublishedAt: string;
}

// CompletenessInfo represents what docs/files exist in a repo.
export interface CompletenessInfo {
	HasDescription: boolean;
	HasReadme: boolean;
	HasLicense: boolean;
	HasTopics: boolean;
	HasPages: boolean;
	HasHomepage: boolean;
	HasProjectJson: boolean;
	HasClaudeMd: boolean;
	HasAgentsMd: boolean;
	[key: string]: boolean;
}

// Repo represents a unified repository from local git and GitHub.
export interface Repo {
	// Identity
	Name: string;
	Description: string;
	Visibility: Visibility;
	HomepageURL: string;
	Topics: string[];
	Language: string;

	// Clone state
	Cloned: boolean;
	LocalPath: string;
	Branch: string;
	Dirty: boolean;
	LocalLastCommit: string;

	// GitHub metadata
	GitHubLastPush: string;
	OpenPRs: number;
	ActionsStatus: ActionsStatus;
	LatestRelease: ReleaseInfo | null;

	// Activity tracking
	NewRelease: boolean;

	// Completeness
	Completeness: CompletenessInfo;

	// Lifecycle classification
	Lifecycle: Lifecycle;
}

// Config represents the CatScan configuration.
export interface Config {
	ScanPath: string;
	GitHubOwner: string;
	Port: number;
	LocalIntervalSeconds: number;
	GitHubIntervalSeconds: number;
	StaleDays: number;
	AbandonedDays: number;
	Notifications: NotificationsConfig;
}

// NotificationsConfig represents notification settings.
export interface NotificationsConfig {
	ActionsChanged: boolean;
	NewRelease: boolean;
	PROpened: boolean;
}

// Health represents the health check response.
export interface Health {
	Uptime: string;
	LastLocalPoll: string;
	LastGitHubPoll: string;
	TotalRepos: number;
	GhAvailable: boolean;
	GhAuthenticated: boolean;
}

// SSE event types from the backend.
export type SSEEventType =
	| "connected"
	| "repos_updated"
	| "github_updated"
	| "actions_changed"
	| "new_release"
	| "pr_opened"
	| "clone_progress"
	| "heartbeat"
	| "error";

// SSEEvent represents a server-sent event.
export interface SSEEvent {
	type: SSEEventType;
	data: unknown;
}

// CloneProgressData represents clone progress event data.
export interface CloneProgressData {
	repo: string;
	state: string;
	error?: string;
}

// ErrorEventData represents error event data.
export interface ErrorEventData {
	type: string;
	error: string;
}

// Filter options for the repo list.
export interface FilterOptions {
	lifecycle?: string;
	visibility?: string;
	cloned?: boolean;
	language?: string;
}

// Sort options for the repo list.
export interface SortOptions {
	field: "name" | "lastUpdate" | "lifecycle";
	order: "asc" | "desc";
}

// Summary statistics for the repo list.
export interface SummaryStats {
	total: number;
	cloned: number;
	public: number;
	private: number;
	ongoing: number;
	maintenance: number;
	stale: number;
	abandoned: number;
}

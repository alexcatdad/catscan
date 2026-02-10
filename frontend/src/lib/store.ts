// Main application store using Svelte 5 runes.

import { onDestroy } from "svelte";
import * as api from "./api";
import { type SSEHandlers, createSSEClient } from "./sse";
import type { FilterOptions, Repo, SortOptions, SummaryStats } from "./types";

// Store state
let repos = $state<Repo[]>([]);
let loading = $state<boolean>(true);
let error = $state<string | null>(null);
let sseConnected = $state<boolean>(false);
let sseClientId = $state<string>("");
let cloneInProgress = $state<Set<string>>(new Set());

// Filter and sort state
let filters = $state<FilterOptions>({});
let sort = $state<SortOptions>({ field: "name", order: "asc" });

// Expanded row state
let expandedRepo = $state<string | null>(null);

// Selection state (for multi-select/clone)
let selectedRepos = $state<Set<string>>(new Set());

// Error banner state
let ghError = $state<{ type: string; message: string } | null>(null);

// Derived: filtered and sorted repos
const filteredRepos = $derived(() => {
	let result = [...repos];

	// Apply filters
	if (filters.lifecycle) {
		result = result.filter((r) => r.Lifecycle === filters.lifecycle);
	}
	if (filters.visibility) {
		result = result.filter((r) => r.Visibility === filters.visibility);
	}
	if (filters.cloned !== undefined) {
		result = result.filter((r) => r.Cloned === filters.cloned);
	}
	if (filters.language) {
		result = result.filter((r) => r.Language === filters.language);
	}

	// Apply sorting
	result.sort((a, b) => {
		let comparison = 0;

		switch (sort.field) {
			case "name":
				comparison = a.Name.localeCompare(b.Name);
				break;
			case "lastUpdate":
				comparison = new Date(a.GitHubLastPush).getTime() - new Date(b.GitHubLastPush).getTime();
				break;
			case "lifecycle":
				comparison = a.Lifecycle.localeCompare(b.Lifecycle);
				break;
		}

		return sort.order === "desc" ? -comparison : comparison;
	});

	return result;
});

// Derived: summary statistics
const summaryStats = $derived(() => {
	const stats: SummaryStats = {
		total: repos.length,
		cloned: 0,
		public: 0,
		private: 0,
		ongoing: 0,
		maintenance: 0,
		stale: 0,
		abandoned: 0,
	};

	for (const repo of repos) {
		if (repo.Cloned) stats.cloned++;
		if (repo.Visibility === "public") stats.public++;
		if (repo.Visibility === "private") stats.private++;
		switch (repo.Lifecycle) {
			case "ongoing":
				stats.ongoing++;
				break;
			case "maintenance":
				stats.maintenance++;
				break;
			case "stale":
				stats.stale++;
				break;
			case "abandoned":
				stats.abandoned++;
				break;
		}
	}

	return stats;
});

// Derived: count of currently filtered repos
const filteredCount = $derived(() => filteredRepos.length);

// Derived: count of selected repos
const selectedCount = $derived(() => selectedRepos.size);

// Derived: selected repos that can be cloned (uncloned repos)
const clonableSelected = $derived(() => {
	return [...selectedRepos].filter((name) => {
		const repo = repos.find((r) => r.Name === name);
		return repo && !repo.Cloned;
	});
});

// Derived: whether any selected repos are clonable
const canClone = $derived(() => clonableSelected.length > 0 && cloneInProgress.size === 0);

// Initialize the store - fetch initial data and set up SSE
export async function initializeStore(): Promise<void> {
	// Fetch initial data
	try {
		const [initialRepos, health] = await Promise.all([
			api.getRepos(),
			api
				.getHealth()
				.catch(() => null), // Health might fail on first load
		]);

		repos = initialRepos;
		loading = false;

		// Check for gh errors
		if (health) {
			if (!health.GhAvailable) {
				showGHError("gh_not_found", "gh CLI not found. Please install gh CLI.");
			} else if (!health.GhAuthenticated) {
				showGHError("gh_auth_error", "gh CLI not authenticated. Please run 'gh auth login'.");
			}
		}
	} catch (err) {
		if (err instanceof api.APIError) {
			error = `Failed to load data: ${err.message}`;
		} else {
			error = "Failed to load data. Is the server running?";
		}
		loading = false;
		return;
	}

	// Set up SSE connection
	setupSSE();
}

// Set up SSE connection and event handlers
function setupSSE(): void {
	const handlers: SSEHandlers = {
		onConnected: (clientId) => {
			sseConnected = true;
			sseClientId = clientId;
		},
		onReposUpdated: (newRepos) => {
			repos = newRepos;
			loading = false;
		},
		onGitHubUpdated: (newRepos) => {
			repos = newRepos;
			loading = false;
		},
		onActionsChanged: (data) => {
			// Update the specific repo's CI status
			updateRepo(data.repo, (repo) => ({
				...repo,
				ActionsStatus: data.newStatus as any,
			}));
		},
		onNewRelease: (data) => {
			// Update the repo's release info
			updateRepo(data.repo, (repo) => ({
				...repo,
				LatestRelease: {
					TagName: data.tagName,
					PublishedAt: data.released,
				},
				NewRelease: true,
			}));
		},
		onPROpened: (data) => {
			// Update the repo's PR count
			updateRepo(data.repo, (repo) => ({
				...repo,
				OpenPRs: data.newCount,
			}));
		},
		onCloneProgress: (data) => {
			if (data.state === "started" || data.state === "cloning") {
				cloneInProgress = new Set(cloneInProgress).add(data.repo);
			} else if (data.state === "complete") {
				// Repo now cloned, trigger full refresh
				const newSet = new Set(cloneInProgress);
				newSet.delete(data.repo);
				cloneInProgress = newSet;
				// The next poll will update the repo list
			} else if (data.state === "error") {
				const newSet = new Set(cloneInProgress);
				newSet.delete(data.repo);
				cloneInProgress = newSet;
				error = `Failed to clone ${data.repo}: ${data.error || "unknown error"}`;
			}
		},
		onError: (data) => {
			if (data.type === "gh_not_found") {
				showGHError("gh_not_found", "gh CLI not found. Please install gh CLI.");
			} else if (data.type === "gh_auth_error") {
				showGHError("gh_auth_error", "gh CLI not authenticated. Please run 'gh auth login'.");
			}
		},
	};

	window.sseClient = createSSEClient(handlers);

	// Cleanup on page unload
	onDestroy(() => {
		window.sseClient?.disconnect();
	});
}

// Helper to update a single repo in the list
function updateRepo(repoName: string, updater: (repo: Repo) => Repo): void {
	repos = repos.map((repo) => (repo.Name === repoName ? updater(repo) : repo));
}

// Show a GitHub CLI error
function showGHError(type: string, message: string): void {
	ghError = { type, message };
}

// Dismiss the GitHub error banner
export function dismissGHError(): void {
	ghError = null;
}

// Actions
export function setFilters(newFilters: FilterOptions): void {
	filters = { ...filters, ...newFilters };
}

export function setSort(newSort: SortOptions): void {
	sort = newSort;
}

export function clearFilters(): void {
	filters = {};
}

export function toggleRepoExpanded(repoName: string): void {
	expandedRepo = expandedRepo === repoName ? null : repoName;
}

export function isRepoExpanded(repoName: string): boolean {
	return expandedRepo === repoName;
}

export function toggleRepoSelection(repoName: string): void {
	if (selectedRepos.has(repoName)) {
		const newSet = new Set(selectedRepos);
		newSet.delete(repoName);
		selectedRepos = newSet;
	} else {
		selectedRepos = new Set(selectedRepos).add(repoName);
	}
}

export function isRepoSelected(repoName: string): boolean {
	return selectedRepos.has(repoName);
}

export function clearSelection(): void {
	selectedRepos = new Set();
}

export async function cloneSelected(): Promise<void> {
	const toClone = clonableSelected();

	for (const repoName of toClone) {
		try {
			await api.cloneRepo(repoName);
		} catch (err) {
			if (err instanceof api.APIError) {
				error = `Failed to start cloning ${repoName}: ${err.message}`;
			}
		}
	}
}

export function isRepoCloning(repoName: string): boolean {
	return cloneInProgress.has(repoName);
}

// Export store state and derived values for components to use
export {
	repos,
	loading,
	error,
	sseConnected,
	sseClientId,
	filters,
	sort,
	expandedRepo,
	selectedRepos,
	ghError,
	cloneInProgress,
	filteredRepos,
	summaryStats,
	filteredCount,
	selectedCount,
	canClone,
};

// Declare for SSE client cleanup
declare global {
	interface Window {
		sseClient?: {
			disconnect(): void;
		};
	}
}

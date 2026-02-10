// Main application store using Svelte 5 runes.
// This file MUST be .svelte.ts for rune compilation.
//
// Pattern: internal $state (prefixed _) for mutation,
// exported getter functions for reactive readonly access from components.
// Svelte 5 modules can't export $state or $derived directly.

import * as api from "./api";
import { type SSEHandlers, createSSEClient } from "./sse";
import type { FilterOptions, Repo, SortOptions, SummaryStats } from "./types";

// --- Internal mutable state ---
let _repos = $state<Repo[]>([]);
let _loading = $state<boolean>(true);
let _error = $state<string | null>(null);
let _sseConnected = $state<boolean>(false);
let _sseClientId = $state<string>("");
let _cloneInProgress = $state<Set<string>>(new Set());
let _filters = $state<FilterOptions>({});
let _sort = $state<SortOptions>({ field: "name", order: "asc" });
let _expandedRepo = $state<string | null>(null);
let _selectedRepos = $state<Set<string>>(new Set());
let _ghError = $state<{ type: string; message: string } | null>(null);

// --- Readonly getters (reactive when called in templates/$derived/$effect) ---
export function repos() { return _repos; }
export function loading() { return _loading; }
export function error() { return _error; }
export function sseConnected() { return _sseConnected; }
export function sseClientId() { return _sseClientId; }
export function cloneInProgress() { return _cloneInProgress; }
export function filters() { return _filters; }
export function sort() { return _sort; }
export function expandedRepo() { return _expandedRepo; }
export function selectedRepos() { return _selectedRepos; }
export function ghError() { return _ghError; }

// --- Computed derived values (internal, exposed via getters) ---

const _filteredRepos = $derived.by(() => {
	let result = [..._repos];

	if (_filters.lifecycle) {
		result = result.filter((r) => r.Lifecycle === _filters.lifecycle);
	}
	if (_filters.visibility) {
		result = result.filter((r) => r.Visibility === _filters.visibility);
	}
	if (_filters.cloned !== undefined) {
		result = result.filter((r) => r.Cloned === _filters.cloned);
	}
	if (_filters.language) {
		result = result.filter((r) => r.Language === _filters.language);
	}

	result.sort((a, b) => {
		let comparison = 0;
		switch (_sort.field) {
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
		return _sort.order === "desc" ? -comparison : comparison;
	});

	return result;
});

const _summaryStats = $derived.by(() => {
	const stats: SummaryStats = {
		total: _repos.length,
		cloned: 0,
		public: 0,
		private: 0,
		ongoing: 0,
		maintenance: 0,
		stale: 0,
		abandoned: 0,
	};

	for (const repo of _repos) {
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

const _clonableSelected = $derived.by(() => {
	return [..._selectedRepos].filter((name) => {
		const repo = _repos.find((r) => r.Name === name);
		return repo && !repo.Cloned;
	});
});

export function filteredRepos() { return _filteredRepos; }
export function summaryStats() { return _summaryStats; }
export function filteredCount() { return _filteredRepos.length; }
export function selectedCount() { return _selectedRepos.size; }
export function canClone() { return _clonableSelected.length > 0 && _cloneInProgress.size === 0; }

// --- Actions ---

export async function initializeStore(): Promise<void> {
	try {
		const [initialRepos, health] = await Promise.all([
			api.getRepos(),
			api.getHealth().catch(() => null),
		]);

		_repos = initialRepos;
		_loading = false;

		if (health) {
			if (!health.GhAvailable) {
				_ghError = { type: "gh_not_found", message: "gh CLI not found. Please install gh CLI." };
			} else if (!health.GhAuthenticated) {
				_ghError = { type: "gh_auth_error", message: "gh CLI not authenticated. Please run 'gh auth login'." };
			}
		}
	} catch (err) {
		if (err instanceof api.APIError) {
			_error = `Failed to load data: ${err.message}`;
		} else {
			_error = "Failed to load data. Is the server running?";
		}
		_loading = false;
		return;
	}

	setupSSE();
}

function setupSSE(): void {
	const handlers: SSEHandlers = {
		onConnected: (clientId) => {
			_sseConnected = true;
			_sseClientId = clientId;
		},
		onReposUpdated: (newRepos) => {
			_repos = newRepos;
			_loading = false;
		},
		onGitHubUpdated: (newRepos) => {
			_repos = newRepos;
			_loading = false;
		},
		onActionsChanged: (data) => {
			_repos = _repos.map((repo) =>
				repo.Name === data.repo ? { ...repo, ActionsStatus: data.newStatus as any } : repo
			);
		},
		onNewRelease: (data) => {
			_repos = _repos.map((repo) =>
				repo.Name === data.repo
					? { ...repo, LatestRelease: { TagName: data.tagName, PublishedAt: data.released }, NewRelease: true }
					: repo
			);
		},
		onPROpened: (data) => {
			_repos = _repos.map((repo) =>
				repo.Name === data.repo ? { ...repo, OpenPRs: data.newCount } : repo
			);
		},
		onCloneProgress: (data) => {
			if (data.state === "started" || data.state === "cloning") {
				_cloneInProgress = new Set(_cloneInProgress).add(data.repo);
			} else if (data.state === "complete") {
				const newSet = new Set(_cloneInProgress);
				newSet.delete(data.repo);
				_cloneInProgress = newSet;
			} else if (data.state === "error") {
				const newSet = new Set(_cloneInProgress);
				newSet.delete(data.repo);
				_cloneInProgress = newSet;
				_error = `Failed to clone ${data.repo}: ${data.error || "unknown error"}`;
			}
		},
		onError: (data) => {
			if (data.type === "gh_not_found") {
				_ghError = { type: "gh_not_found", message: "gh CLI not found. Please install gh CLI." };
			} else if (data.type === "gh_auth_error") {
				_ghError = { type: "gh_auth_error", message: "gh CLI not authenticated. Please run 'gh auth login'." };
			}
		},
	};

	window.sseClient = createSSEClient(handlers);

	// Cleanup SSE on page unload (not onDestroy — this isn't a component)
	window.addEventListener("beforeunload", () => {
		window.sseClient?.disconnect();
	});
}

export function dismissGHError(): void {
	_ghError = null;
}

export function setFilters(newFilters: FilterOptions): void {
	_filters = { ..._filters, ...newFilters };
}

export function setSort(newSort: SortOptions): void {
	_sort = newSort;
}

export function clearFilters(): void {
	_filters = {};
}

export function toggleRepoExpanded(repoName: string): void {
	_expandedRepo = _expandedRepo === repoName ? null : repoName;
}

export function isRepoExpanded(repoName: string): boolean {
	return _expandedRepo === repoName;
}

export function toggleRepoSelection(repoName: string): void {
	if (_selectedRepos.has(repoName)) {
		const newSet = new Set(_selectedRepos);
		newSet.delete(repoName);
		_selectedRepos = newSet;
	} else {
		_selectedRepos = new Set(_selectedRepos).add(repoName);
	}
}

export function isRepoSelected(repoName: string): boolean {
	return _selectedRepos.has(repoName);
}

export function clearSelection(): void {
	_selectedRepos = new Set();
}

export async function cloneSelected(): Promise<void> {
	for (const repoName of _clonableSelected) {
		try {
			await api.cloneRepo(repoName);
		} catch (err) {
			if (err instanceof api.APIError) {
				_error = `Failed to start cloning ${repoName}: ${err.message}`;
			}
		}
	}
}

export function isRepoCloning(repoName: string): boolean {
	return _cloneInProgress.has(repoName);
}

// Declare for SSE client cleanup
declare global {
	interface Window {
		sseClient?: {
			disconnect(): void;
		};
	}
}

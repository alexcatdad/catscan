// API client for the CatScan backend.

import type { Config, FilterOptions, Health, Repo, SortOptions } from "./types";

const API_BASE = "/api";

// APIError represents an error response from the backend.
export class APIError extends Error {
	status: number;
	constructor(message: string, status: number) {
		super(message);
		this.name = "APIError";
		this.status = status;
	}
}

// Helper to make fetch requests and throw on non-2xx status codes.
async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
	const response = await fetch(url, options);

	if (!response.ok) {
		let message = `HTTP ${response.status}`;
		try {
			const data = await response.json();
			if (data.error) {
				message = data.error;
			}
		} catch {
			// Use default message
		}
		throw new APIError(message, response.status);
	}

	return response.json() as Promise<T>;
}

// Get repos from the backend with optional filtering and sorting.
export async function getRepos(filters?: FilterOptions, sort?: SortOptions): Promise<Repo[]> {
	const params = new URLSearchParams();

	if (filters?.lifecycle) {
		params.set("lifecycle", filters.lifecycle);
	}
	if (filters?.visibility) {
		params.set("visibility", filters.visibility);
	}
	if (filters?.cloned !== undefined) {
		params.set("cloned", String(filters.cloned));
	}
	if (filters?.language) {
		params.set("language", filters.language);
	}
	if (sort?.field) {
		params.set("sort", sort.field);
		params.set("order", sort.order);
	}

	const query = params.toString();
	return fetchJSON<Repo[]>(`${API_BASE}/repos${query ? `?${query}` : ""}`);
}

// Get a single repo by name.
export async function getRepo(name: string): Promise<Repo> {
	return fetchJSON<Repo>(`${API_BASE}/repos/${encodeURIComponent(name)}`);
}

// Start cloning a repo.
export async function cloneRepo(name: string): Promise<{ status: string }> {
	return fetchJSON<{ status: string }>(`${API_BASE}/repos/${encodeURIComponent(name)}/clone`, {
		method: "POST",
	});
}

// Get the current config.
export async function getConfig(): Promise<Config> {
	return fetchJSON<Config>(`${API_BASE}/config`);
}

// Update the config.
export async function updateConfig(config: Config): Promise<Config> {
	return fetchJSON<Config>(`${API_BASE}/config`, {
		method: "PUT",
		headers: {
			"Content-Type": "application/json",
		},
		body: JSON.stringify(config),
	});
}

// Get health status.
export async function getHealth(): Promise<Health> {
	return fetchJSON<Health>(`${API_BASE}/health`);
}

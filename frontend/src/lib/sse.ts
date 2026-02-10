// SSE client for real-time updates from the CatScan backend.

import type { Repo, SSEEventType } from "./types";

// Event handlers for SSE events.
export interface SSEHandlers {
	onConnected?: (clientId: string) => void;
	onReposUpdated?: (repos: Repo[]) => void;
	onGitHubUpdated?: (repos: Repo[]) => void;
	onActionsChanged?: (data: {
		repo: string;
		oldStatus: string;
		newStatus: string;
	}) => void;
	onNewRelease?: (data: {
		repo: string;
		tagName: string;
		released: string;
	}) => void;
	onPROpened?: (data: {
		repo: string;
		oldCount: number;
		newCount: number;
	}) => void;
	onCloneProgress?: (data: {
		repo: string;
		state: string;
		error?: string;
	}) => void;
	onHeartbeat?: () => void;
	onError?: (data: { type: string; error: string }) => void;
}

// Reconnection configuration.
interface ReconnectConfig {
	baseDelay: number;
	maxDelay: number;
	backoffMultiplier: number;
}

const DEFAULT_RECONNECT: ReconnectConfig = {
	baseDelay: 1000, // 1 second
	maxDelay: 30000, // 30 seconds
	backoffMultiplier: 2,
};

// SSEClient manages a Server-Sent Events connection.
export class SSEClient {
	private eventSource: EventSource | null = null;
	private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	private reconnectDelay = DEFAULT_RECONNECT.baseDelay;
	private reconnectConfig: ReconnectConfig;
	private handshakeReceived = false;

	constructor(
		private url: string,
		private handlers: SSEHandlers,
		reconnectConfig?: Partial<ReconnectConfig>
	) {
		this.reconnectConfig = { ...DEFAULT_RECONNECT, ...reconnectConfig };
	}

	// Connect to the SSE endpoint.
	connect(): void {
		if (this.eventSource) {
			return; // Already connected
		}

		this.eventSource = new EventSource(this.url);

		this.eventSource.onopen = () => {
			// Connection opened
			this.reconnectDelay = this.reconnectConfig.baseDelay;
		};

		this.eventSource.onerror = (event) => {
			// Connection error or closed
			if (this.eventSource?.readyState === EventSource.CLOSED) {
				this.scheduleReconnect();
			}
		};

		// Listen for all message types
		const eventTypes: SSEEventType[] = [
			"connected",
			"repos_updated",
			"github_updated",
			"actions_changed",
			"new_release",
			"pr_opened",
			"clone_progress",
			"heartbeat",
			"error",
		];

		for (const eventType of eventTypes) {
			this.eventSource.addEventListener(eventType, (e) => {
				this.handleEvent(eventType, e);
			});
		}
	}

	// Disconnect from the SSE endpoint.
	disconnect(): void {
		if (this.reconnectTimer) {
			clearTimeout(this.reconnectTimer);
			this.reconnectTimer = null;
		}

		if (this.eventSource) {
			this.eventSource.close();
			this.eventSource = null;
		}

		this.handshakeReceived = false;
	}

	// Schedule a reconnection attempt with exponential backoff.
	private scheduleReconnect(): void {
		if (this.reconnectTimer) {
			return; // Already scheduled
		}

		this.reconnectTimer = setTimeout(() => {
			this.reconnectTimer = null;
			this.connect();
		}, this.reconnectDelay);

		// Exponential backoff
		this.reconnectDelay = Math.min(
			this.reconnectDelay * this.reconnectConfig.backoffMultiplier,
			this.reconnectConfig.maxDelay
		);
	}

	// Handle an incoming SSE event.
	private handleEvent(eventType: SSEEventType, e: MessageEvent): void {
		try {
			const data = JSON.parse(e.data);

			switch (eventType) {
				case "connected":
					this.handshakeReceived = true;
					this.handlers.onConnected?.(data.clientId);
					break;
				case "repos_updated":
				case "github_updated":
					// Type guard to ensure data is Repo[]
					if (Array.isArray(data)) {
						if (eventType === "repos_updated") {
							this.handlers.onReposUpdated?.(data as Repo[]);
						} else {
							this.handlers.onGitHubUpdated?.(data as Repo[]);
						}
					}
					break;
				case "actions_changed":
					this.handlers.onActionsChanged?.(data);
					break;
				case "new_release":
					this.handlers.onNewRelease?.(data);
					break;
				case "pr_opened":
					this.handlers.onPROpened?.(data);
					break;
				case "clone_progress":
					this.handlers.onCloneProgress?.(data);
					break;
				case "heartbeat":
					this.handlers.onHeartbeat?.();
					break;
				case "error":
					this.handlers.onError?.(data);
					break;
			}
		} catch (error) {
			console.error("Failed to parse SSE event:", error, e.data);
		}
	}
}

// Create and connect an SSE client for the CatScan events endpoint.
export function createSSEClient(handlers: SSEHandlers): SSEClient {
	const client = new SSEClient("/api/events", handlers);
	client.connect();
	return client;
}

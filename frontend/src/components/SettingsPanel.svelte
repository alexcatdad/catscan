<!-- Settings panel modal -->
<script lang="ts">
	import { XIcon } from "svelte-feather-icons";
	import type { Config } from "$lib/types";
	import { getConfig, updateConfig } from "$lib/api";

	interface Props {
		show: boolean;
		onClose: () => void;
	}

	let { show, onClose }: Props = $props();

	let config = $state<Config | null>(null);
	let loading = $state(false);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let success = $state(false);

	// Form state
	let scanPath = $state("");
	let gitHubOwner = $state("");
	let port = $state(8080);
	let localIntervalSeconds = $state(30);
	let gitHubIntervalSeconds = $state(300);
	let staleDays = $state(90);
	let abandonedDays = $state(365);
	let actionsChanged = $state(false);
	let newRelease = $state(false);
	let pROpened = $state(false);

	async function loadConfig() {
		loading = true;
		error = null;
		try {
			const cfg = await getConfig();
			config = cfg;
			scanPath = cfg.scanPath;
			gitHubOwner = cfg.githubOwner;
			port = cfg.port;
			localIntervalSeconds = cfg.localIntervalSeconds;
			gitHubIntervalSeconds = cfg.githubIntervalSeconds;
			staleDays = cfg.staleDays;
			abandonedDays = cfg.abandonedDays;
			actionsChanged = cfg.notifications.actionsChanged;
			newRelease = cfg.notifications.newRelease;
			pROpened = cfg.notifications.prOpened;
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to load config";
		} finally {
			loading = false;
		}
	}

	// Load config when panel opens
	$effect(() => {
		if (show) {
			loadConfig();
			success = false;
		}
	});

	async function handleSave() {
		const validationErrors = validate();
		if (validationErrors.length > 0) {
			error = validationErrors.join("; ");
			return;
		}

		saving = true;
		error = null;
		try {
			const newConfig: Config = {
				scanPath,
				githubOwner: gitHubOwner,
				port,
				localIntervalSeconds,
				githubIntervalSeconds: gitHubIntervalSeconds,
				staleDays,
				abandonedDays,
				notifications: {
					actionsChanged,
					newRelease,
					prOpened: pROpened,
					cloneCompleted: config?.notifications.cloneCompleted ?? true,
					error: config?.notifications.error ?? true,
				},
			};
			await updateConfig(newConfig);
			success = true;
			setTimeout(() => {
				onClose();
			}, 500);
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to save config";
		} finally {
			saving = false;
		}
	}

	function validate(): string[] {
		const errors: string[] = [];

		if (!scanPath?.trim()) {
			errors.push("Scan path is required");
		}
		if (port < 1024 || port > 65535) {
			errors.push("Port must be between 1024 and 65535");
		}
		if (localIntervalSeconds < 10) {
			errors.push("Local poll interval must be at least 10 seconds");
		}
		if (gitHubIntervalSeconds < 60) {
			errors.push("GitHub poll interval must be at least 60 seconds");
		}
		if (staleDays < 1) {
			errors.push("Stale days must be at least 1");
		}
		if (abandonedDays < 1) {
			errors.push("Abandoned days must be at least 1");
		}
		if (staleDays >= abandonedDays) {
			errors.push("Stale threshold must be less than abandoned threshold");
		}

		return errors;
	}

	function handleClose() {
		if (!saving) {
			onClose();
		}
	}
</script>

{#if show}
	<div class="settings-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onclick={handleClose}>
		<div
			class="settings-modal w-full max-w-lg rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-surface)] p-6 shadow-2xl shadow-black/40"
			onclick={(e) => e.stopPropagation()}
			data-testid="settings-panel"
		>
			<div class="mb-5 flex items-center justify-between">
				<h2 class="font-[var(--font-mono)] text-base font-medium text-[var(--color-fg-base)]">Settings</h2>
				<button
					onclick={handleClose}
					class="rounded-lg p-1.5 text-[var(--color-fg-subtle)] transition-colors hover:bg-[var(--color-bg-elevated)] hover:text-[var(--color-fg-base)] disabled:opacity-50"
					disabled={saving}
					aria-label="Close"
				>
					<XIcon size="18" />
				</button>
			</div>

			{#if loading}
				<div class="flex items-center justify-center py-10">
					<div class="h-6 w-6 animate-spin rounded-full border-2 border-[var(--color-border)] border-t-[var(--color-accent)]"></div>
				</div>
			{:else if config}
				<form onsubmit={(e) => { e.preventDefault(); handleSave(); }} class="space-y-5">
					<!-- Scan Path -->
					<div>
						<label for="scanPath" class="settings-label">Scan Path</label>
						<input
							id="scanPath"
							type="text"
							bind:value={scanPath}
							class="settings-input"
							placeholder="/path/to/repos"
						/>
						<p class="mt-1.5 text-xs text-[var(--color-fg-subtle)]">
							Local directory containing your repositories
						</p>
					</div>

					<!-- GitHub Owner -->
					<div>
						<label for="gitHubOwner" class="settings-label">GitHub Owner</label>
						<input
							id="gitHubOwner"
							type="text"
							bind:value={gitHubOwner}
							class="settings-input"
							placeholder="username or org"
						/>
					</div>

					<!-- Port -->
					<div>
						<label for="port" class="settings-label">Port</label>
						<input
							id="port"
							type="number"
							min="1024"
							max="65535"
							bind:value={port}
							class="settings-input"
						/>
					</div>

					<!-- Poll Intervals -->
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="localInterval" class="settings-label">Local Poll (sec)</label>
							<input
								id="localInterval"
								type="number"
								min="10"
								bind:value={localIntervalSeconds}
								class="settings-input"
							/>
						</div>
						<div>
							<label for="gitHubInterval" class="settings-label">GitHub Poll (sec)</label>
							<input
								id="gitHubInterval"
								type="number"
								min="60"
								bind:value={gitHubIntervalSeconds}
								class="settings-input"
							/>
						</div>
					</div>

					<!-- Lifecycle Thresholds -->
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="staleDays" class="settings-label">Stale (days)</label>
							<input
								id="staleDays"
								type="number"
								min="1"
								bind:value={staleDays}
								class="settings-input"
							/>
						</div>
						<div>
							<label for="abandonedDays" class="settings-label">Abandoned (days)</label>
							<input
								id="abandonedDays"
								type="number"
								min="1"
								bind:value={abandonedDays}
								class="settings-input"
							/>
						</div>
					</div>

					<!-- Notifications -->
					<div class="space-y-3 border-t border-[var(--color-border-subtle)] pt-4">
						<h3 class="font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Notifications</h3>
						<div class="space-y-2">
							<label class="flex items-center gap-3 rounded-lg px-2 py-1.5 transition-colors hover:bg-[var(--color-bg-elevated)]">
								<input
									type="checkbox"
									bind:checked={actionsChanged}
									class="h-4 w-4 rounded border-[var(--color-border)] bg-[var(--color-bg-base)] accent-[var(--color-accent)]"
								/>
								<span class="text-sm text-[var(--color-fg-base)]">CI status changes</span>
							</label>
							<label class="flex items-center gap-3 rounded-lg px-2 py-1.5 transition-colors hover:bg-[var(--color-bg-elevated)]">
								<input
									type="checkbox"
									bind:checked={newRelease}
									class="h-4 w-4 rounded border-[var(--color-border)] bg-[var(--color-bg-base)] accent-[var(--color-accent)]"
								/>
								<span class="text-sm text-[var(--color-fg-base)]">New releases</span>
							</label>
							<label class="flex items-center gap-3 rounded-lg px-2 py-1.5 transition-colors hover:bg-[var(--color-bg-elevated)]">
								<input
									type="checkbox"
									bind:checked={pROpened}
									class="h-4 w-4 rounded border-[var(--color-border)] bg-[var(--color-bg-base)] accent-[var(--color-accent)]"
								/>
								<span class="text-sm text-[var(--color-fg-base)]">New pull requests</span>
							</label>
						</div>
						<p class="px-2 text-xs text-[var(--color-fg-subtle)]">
							Sent via macOS Notification Center (terminal-notifier or osascript required)
						</p>
					</div>

					<!-- Error Message -->
					{#if error}
						<div class="rounded-lg border border-[var(--color-error)]/20 bg-[var(--color-error)]/5 px-3 py-2">
							<p class="text-sm text-[var(--color-error)]">{error}</p>
						</div>
					{/if}

					<!-- Success Message -->
					{#if success}
						<div class="rounded-lg border border-[var(--color-success)]/20 bg-[var(--color-success)]/5 px-3 py-2">
							<p class="text-sm text-[var(--color-success)]">Settings saved</p>
						</div>
					{/if}

					<!-- Actions -->
					<div class="flex justify-end gap-2 pt-2">
						<button
							type="button"
							onclick={handleClose}
							disabled={saving}
							class="rounded-lg border border-[var(--color-border)] px-4 py-2 text-sm text-[var(--color-fg-muted)] transition-colors hover:bg-[var(--color-bg-elevated)] hover:text-[var(--color-fg-base)] disabled:opacity-50"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={saving}
							class="rounded-lg bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-[var(--color-bg-base)] shadow-[0_0_12px_oklch(0.72_0.14_192/0.15)] transition-all hover:bg-[var(--color-accent-hover)] disabled:opacity-50 disabled:shadow-none"
						>
							{saving ? "Saving..." : "Save"}
						</button>
					</div>
				</form>
			{/if}
		</div>
	</div>
{/if}

<style>
	.settings-overlay {
		animation: fade-in 0.15s ease-out;
	}

	.settings-modal {
		animation: slide-up 0.2s ease-out;
	}

	.settings-label {
		display: block;
		margin-bottom: 4px;
		font-family: "Azeret Mono", ui-monospace, monospace;
		font-size: 11px;
		font-weight: 500;
		letter-spacing: 0.05em;
		color: oklch(0.58 0.018 260);
	}

	.settings-input {
		width: 100%;
		border-radius: 8px;
		border: 1px solid oklch(0.24 0.015 265);
		background: oklch(0.12 0.02 265);
		padding: 8px 12px;
		font-family: "Azeret Mono", ui-monospace, monospace;
		font-size: 13px;
		color: oklch(0.92 0.008 250);
		transition: border-color 0.15s, box-shadow 0.15s;
	}

	.settings-input:focus {
		outline: none;
		border-color: oklch(0.72 0.14 192);
		box-shadow: 0 0 0 3px oklch(0.72 0.14 192 / 0.1);
	}

	.settings-input::placeholder {
		color: oklch(0.38 0.015 260);
	}
</style>

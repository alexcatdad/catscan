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
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onclick={handleClose}>
		<div
			class="w-full max-w-lg rounded-md border border-[var(--color-border)] bg-[var(--color-bg-surface)] p-6 shadow-lg"
			onclick={(e) => e.stopPropagation()}
			data-testid="settings-panel"
		>
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold text-[var(--color-fg-base)]">Settings</h2>
				<button
					onclick={handleClose}
					class="text-[var(--color-fg-muted)] hover:text-[var(--color-fg-base)] disabled:opacity-50"
					disabled={saving}
					aria-label="Close"
				>
					<XIcon size="20" />
				</button>
			</div>

			{#if loading}
				<div class="flex items-center justify-center py-8">
					<div class="h-6 w-6 animate-spin rounded-full border-2 border-[var(--color-fg-muted)] border-t-transparent"></div>
				</div>
			{:else if config}
				<form onsubmit={(e) => { e.preventDefault(); handleSave(); }} class="space-y-4">
					<!-- Scan Path -->
					<div>
						<label for="scanPath" class="mb-1 block text-sm font-medium text-[var(--color-fg-base)]"
							>Scan Path</label
						>
						<input
							id="scanPath"
							type="text"
							bind:value={scanPath}
							class="w-full rounded-md border border-[var(--color-border)] bg-[var(--color-bg-base)] px-3 py-2 text-sm text-[var(--color-fg-base)] focus:border-[var(--color-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
							placeholder="/path/to/repos"
						/>
						<p class="mt-1 text-xs text-[var(--color-fg-muted)]">
							Local directory containing your repositories
						</p>
					</div>

					<!-- GitHub Owner -->
					<div>
						<label for="gitHubOwner" class="mb-1 block text-sm font-medium text-[var(--color-fg-base)]"
							>GitHub Owner</label
						>
						<input
							id="gitHubOwner"
							type="text"
							bind:value={gitHubOwner}
							class="w-full rounded-md border border-[var(--color-border)] bg-[var(--color-bg-base)] px-3 py-2 text-sm text-[var(--color-fg-base)] focus:border-[var(--color-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
							placeholder="username or org"
						/>
						<p class="mt-1 text-xs text-[var(--color-fg-muted)]">
							GitHub username or organization to scan
						</p>
					</div>

					<!-- Port -->
					<div>
						<label for="port" class="mb-1 block text-sm font-medium text-[var(--color-fg-base)]">Port</label>
						<input
							id="port"
							type="number"
							min="1024"
							max="65535"
							bind:value={port}
							class="w-full rounded-md border border-[var(--color-border)] bg-[var(--color-bg-base)] px-3 py-2 text-sm text-[var(--color-fg-base)] focus:border-[var(--color-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
						/>
					</div>

					<!-- Poll Intervals -->
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label
								for="localInterval"
								class="mb-1 block text-sm font-medium text-[var(--color-fg-base)]"
								>Local Poll Interval (sec)</label
							>
							<input
								id="localInterval"
								type="number"
								min="10"
								bind:value={localIntervalSeconds}
								class="w-full rounded-md border border-[var(--color-border)] bg-[var(--color-bg-base)] px-3 py-2 text-sm text-[var(--color-fg-base)] focus:border-[var(--color-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
							/>
						</div>
						<div>
							<label
								for="gitHubInterval"
								class="mb-1 block text-sm font-medium text-[var(--color-fg-base)]"
								>GitHub Poll Interval (sec)</label
							>
							<input
								id="gitHubInterval"
								type="number"
								min="60"
								bind:value={gitHubIntervalSeconds}
								class="w-full rounded-md border border-[var(--color-border)] bg-[var(--color-bg-base)] px-3 py-2 text-sm text-[var(--color-fg-base)] focus:border-[var(--color-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
							/>
						</div>
					</div>

					<!-- Lifecycle Thresholds -->
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="staleDays" class="mb-1 block text-sm font-medium text-[var(--color-fg-base)]"
								>Stale Threshold (days)</label
							>
							<input
								id="staleDays"
								type="number"
								min="1"
								bind:value={staleDays}
								class="w-full rounded-md border border-[var(--color-border)] bg-[var(--color-bg-base)] px-3 py-2 text-sm text-[var(--color-fg-base)] focus:border-[var(--color-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
							/>
						</div>
						<div>
							<label
								for="abandonedDays"
								class="mb-1 block text-sm font-medium text-[var(--color-fg-base)]"
								>Abandoned Threshold (days)</label
							>
							<input
								id="abandonedDays"
								type="number"
								min="1"
								bind:value={abandonedDays}
								class="w-full rounded-md border border-[var(--color-border)] bg-[var(--color-bg-base)] px-3 py-2 text-sm text-[var(--color-fg-base)] focus:border-[var(--color-accent)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
							/>
						</div>
					</div>

					<!-- Notifications -->
					<div class="space-y-2">
						<h3 class="text-sm font-medium text-[var(--color-fg-base)]">Notifications</h3>
						<div class="space-y-2">
							<label class="flex items-center gap-2">
								<input
									type="checkbox"
									bind:checked={actionsChanged}
									class="h-4 w-4 rounded border-[var(--color-border)] bg-[var(--color-bg-base)] accent-[var(--color-accent)]"
								/>
								<span class="text-sm text-[var(--color-fg-base)]">CI status changes</span>
							</label>
							<label class="flex items-center gap-2">
								<input
									type="checkbox"
									bind:checked={newRelease}
									class="h-4 w-4 rounded border-[var(--color-border)] bg-[var(--color-bg-base)] accent-[var(--color-accent)]"
								/>
								<span class="text-sm text-[var(--color-fg-base)]">New releases</span>
							</label>
							<label class="flex items-center gap-2">
								<input
									type="checkbox"
									bind:checked={pROpened}
									class="h-4 w-4 rounded border-[var(--color-border)] bg-[var(--color-bg-base)] accent-[var(--color-accent)]"
								/>
								<span class="text-sm text-[var(--color-fg-base)]">New pull requests</span>
							</label>
						</div>
						<p class="text-xs text-[var(--color-fg-muted)]">
							Notifications are sent via macOS Notification Center. Terminal-notifier or osascript is
							required.
						</p>
					</div>

					<!-- Error Message -->
					{#if error}
						<div class="rounded-md bg-[var(--color-error)]/10 border border-[var(--color-error)]/20 px-3 py-2">
							<p class="text-sm text-[var(--color-error)]">{error}</p>
						</div>
					{/if}

					<!-- Success Message -->
					{#if success}
						<div class="rounded-md bg-[var(--color-success)]/10 border border-[var(--color-success)]/20 px-3 py-2">
							<p class="text-sm text-[var(--color-success)]">Settings saved successfully!</p>
						</div>
					{/if}

					<!-- Actions -->
					<div class="flex justify-end gap-2 pt-4">
						<button
							type="button"
							onclick={handleClose}
							disabled={saving}
							class="rounded-md border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-fg-base)] hover:bg-[var(--color-bg-elevated)] disabled:opacity-50"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={saving}
							class="rounded-md bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-[var(--color-bg-base)] hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
						>
							{saving ? "Saving..." : "Save Settings"}
						</button>
					</div>
				</form>
			{/if}
		</div>
	</div>
{/if}

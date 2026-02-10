<!-- Main repository table with filtering, sorting, and row expansion -->
<script lang="ts">
	import RepoDetail from "./RepoDetail.svelte";
	import {
		CheckIcon,
		GithubIcon,
		AlertCircleIcon,
		FileTextIcon,
		TerminalIcon,
		DownloadIcon
	} from "svelte-feather-icons";
	import type { Repo } from "$lib/types";
	import {
		filteredRepos,
		toggleRepoExpanded,
		isRepoExpanded,
		toggleRepoSelection,
		isRepoSelected,
		selectedRepos,
		isRepoCloning,
		cloneSelected,
		canClone,
	} from "$lib/store.svelte";
	import { formatRelativeTime, getLifecycleColor, getVisibilityColor, getCIStatusColor, getCompletenessIssues } from "$lib/utils";

	let tableBody = $state<HTMLTableSectionElement>();

	// Handle keyboard navigation
	let focusedIndex = $state<number>(-1);

	function handleKeydown(event: KeyboardEvent, index: number) {
		const repos = filteredRepos();
		const repo = repos[index];
		if (!repo) return;

		switch (event.key) {
			case "ArrowDown":
				event.preventDefault();
				if (index < repos.length - 1) {
					focusedIndex = index + 1;
				}
				break;
			case "ArrowUp":
				event.preventDefault();
				if (index > 0) {
					focusedIndex = index - 1;
				} else {
					focusedIndex = -1;
				}
				break;
			case "Enter":
				event.preventDefault();
				toggleRepoExpanded(repo.Name);
				break;
			case " ":
				event.preventDefault();
				toggleRepoSelection(repo.Name);
				break;
		}
	}

	function focusRow(index: number) {
		focusedIndex = index;
	}
</script>

<div class="table-container overflow-hidden rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-surface)]/40">
	<table class="w-full border-separate border-spacing-0" data-testid="repo-table">
		<thead>
			<tr>
				<th class="w-10 border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]/80 px-4 py-3 text-left backdrop-blur-sm">
					<input
						type="checkbox"
						disabled
						class="pointer-events-none opacity-0"
						aria-label="Select all"
					/>
				</th>
				<th class="border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]/80 px-4 py-3 text-left backdrop-blur-sm">
					<span class="font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Repository</span>
				</th>
				<th class="hidden border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]/80 px-4 py-3 text-left backdrop-blur-sm md:table-cell">
					<span class="font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Visibility</span>
				</th>
				<th class="hidden border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]/80 px-4 py-3 text-left backdrop-blur-sm lg:table-cell">
					<span class="font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Branch</span>
				</th>
				<th class="border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]/80 px-4 py-3 text-left backdrop-blur-sm">
					<span class="font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">CI</span>
				</th>
				<th class="hidden border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]/80 px-4 py-3 text-left backdrop-blur-sm sm:table-cell">
					<span class="font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Complete</span>
				</th>
				<th class="hidden border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]/80 px-4 py-3 text-left backdrop-blur-sm sm:table-cell">
					<span class="font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Updated</span>
				</th>
				<th class="border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]/80 px-4 py-3 text-left backdrop-blur-sm">
					<span class="font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Status</span>
				</th>
			</tr>
		</thead>
		<tbody bind:this={tableBody}>
			{#each filteredRepos() as repo, index}
				{@const isExpanded = isRepoExpanded(repo.Name)}
				{@const isSelected = isRepoSelected(repo.Name)}
				{@const isCloning = isRepoCloning(repo.Name)}
				{@const issues = getCompletenessIssues(repo.Completeness)}
				{@const isFocused = focusedIndex === index}

				<tr
					class="repo-row group border-b border-[var(--color-border-subtle)] transition-colors
						{isFocused ? 'bg-[var(--color-bg-elevated)]/60' : ''}
						{isSelected ? 'bg-[var(--color-accent)]/5' : ''}
						{isExpanded ? 'bg-[var(--color-accent)]/3' : ''}"
					onclick={() => focusRow(index)}
					onkeydown={(e) => handleKeydown(e, index)}
					tabindex={isFocused ? 0 : -1}
					data-testid="repo-row"
				>
					<!-- Checkbox -->
					<td class="px-4 py-3">
						{#if repo.Cloned}
							<div class="h-4 w-4"></div>
						{:else}
							<input
								type="checkbox"
								checked={isSelected}
								onchange={() => toggleRepoSelection(repo.Name)}
								onclick={(e) => e.stopPropagation()}
								class="h-4 w-4 rounded border-[var(--color-border)] bg-[var(--color-bg-base)] accent-[var(--color-accent)]"
								aria-label="Select {repo.Name}"
							/>
						{/if}
					</td>

					<!-- Repo name -->
					<td class="max-w-[220px] px-4 py-3">
						<button
							onclick={() => toggleRepoExpanded(repo.Name)}
							class="repo-name inline-flex items-center gap-2 rounded-md px-1.5 py-0.5 -ml-1.5 transition-colors hover:bg-[var(--color-accent)]/10"
							data-testid="repo-name"
						>
							{#if isCloning}
								<span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-[var(--color-fg-muted)] border-t-[var(--color-accent)]"></span>
							{/if}
							<span class="truncate font-[var(--font-mono)] text-sm font-medium text-[var(--color-accent)]">{repo.Name}</span>
						</button>
						{#if repo.Language}
							<span class="ml-1.5 text-xs text-[var(--color-fg-subtle)]">{repo.Language}</span>
						{/if}
					</td>

					<!-- Visibility -->
					<td class="hidden px-4 py-3 md:table-cell">
						<span class="font-[var(--font-mono)] text-xs text-[var(--color-fg-subtle)]">
							{repo.Visibility}
						</span>
					</td>

					<!-- Branch -->
					<td class="hidden px-4 py-3 lg:table-cell">
						{#if repo.Cloned}
							<div class="flex items-center gap-1.5">
								<TerminalIcon size="11" class="text-[var(--color-fg-subtle)]" />
								<span class="font-[var(--font-mono)] text-xs text-[var(--color-fg-muted)]">{repo.Branch}</span>
								{#if repo.Dirty}
									<span class="inline-block h-1.5 w-1.5 rounded-full bg-[var(--color-warning)]" title="Dirty"></span>
								{/if}
							</div>
						{:else}
							<span class="text-xs text-[var(--color-fg-subtle)]">—</span>
						{/if}
					</td>

					<!-- CI Status -->
					<td class="px-4 py-3">
						<div
							class="ci-dot h-2 w-2 rounded-full {getCIStatusColor(repo.ActionsStatus)}"
							title={repo.ActionsStatus}
						></div>
					</td>

					<!-- Completeness -->
					<td class="hidden px-4 py-3 sm:table-cell">
						{#if issues.length > 0}
							<div class="flex items-center gap-1.5" title={`Missing: ${issues.join(", ")}`}>
								{#if issues.includes("description")}
									<AlertCircleIcon size="13" class="text-[var(--color-warning)]/70" />
								{/if}
								{#if issues.includes("README")}
									<FileTextIcon size="13" class="text-[var(--color-warning)]/70" />
								{/if}
								{#if issues.length > 2}
									<span class="font-[var(--font-mono)] text-[10px] text-[var(--color-fg-subtle)]">+{issues.length - 2}</span>
								{/if}
							</div>
						{:else}
							<span class="font-[var(--font-mono)] text-xs text-[var(--color-success)]">OK</span>
						{/if}
					</td>

					<!-- Last update -->
					<td class="hidden px-4 py-3 sm:table-cell">
						<span class="font-[var(--font-mono)] text-xs tabular-nums text-[var(--color-fg-subtle)]">
							{formatRelativeTime(repo.GitHubLastPush)}
						</span>
					</td>

					<!-- Lifecycle badge -->
					<td class="px-4 py-3">
						<span class="lifecycle-badge inline-flex items-center gap-1.5 font-[var(--font-mono)] text-[11px] uppercase tracking-wider {getLifecycleColor(repo.Lifecycle)}">
							<span class="inline-block h-1.5 w-1.5 rounded-full bg-current"></span>
							{repo.Lifecycle}
						</span>
					</td>
				</tr>

				<!-- Expanded row detail -->
				{#if isExpanded}
					<tr class="border-b border-[var(--color-border-subtle)]">
						<td colspan="8" class="bg-[var(--color-bg-base)]/50 p-0">
							<div class="detail-panel p-5">
								<RepoDetail {repo} />
							</div>
						</td>
					</tr>
				{/if}
			{/each}

			{#if filteredRepos().length === 0}
				<tr>
					<td colspan="8" class="px-4 py-16 text-center">
						<p class="font-[var(--font-mono)] text-sm text-[var(--color-fg-muted)]">No repositories found</p>
						<p class="mt-2 text-xs text-[var(--color-fg-subtle)]">Try adjusting your filters</p>
					</td>
				</tr>
			{/if}
		</tbody>
	</table>
</div>

<!-- Action bar for cloning -->
{#if selectedRepos().size > 0}
	<div class="clone-bar fixed bottom-0 left-0 right-0 z-40 border-t border-[var(--color-border)] bg-[var(--color-bg-surface)]/90 px-6 py-3 shadow-2xl shadow-black/40 backdrop-blur-md">
		<div class="mx-auto flex max-w-[1400px] items-center justify-between">
			<div class="font-[var(--font-mono)] text-sm text-[var(--color-fg-muted)]">
				<span class="text-[var(--color-accent)]">{selectedRepos().size}</span>
				repo{selectedRepos().size === 1 ? "" : "s"} selected
			</div>
			<button
				onclick={cloneSelected}
				disabled={!canClone()}
				class="flex items-center gap-2 rounded-lg bg-[var(--color-accent)] px-4 py-2 font-[var(--font-mono)] text-sm font-medium text-[var(--color-bg-base)] shadow-[0_0_20px_oklch(0.72_0.14_192/0.2)] transition-all hover:bg-[var(--color-accent-hover)] disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none"
			>
				<DownloadIcon size="14" />
				Clone
			</button>
		</div>
	</div>
{/if}

<style>
	.repo-row:hover {
		background-color: oklch(0.16 0.018 265 / 0.6);
	}

	.repo-row:hover .repo-name span {
		text-decoration: none;
	}

	.ci-dot[class*="bg-[var(--color-success)]"] {
		box-shadow: 0 0 6px oklch(0.68 0.16 155 / 0.4);
	}

	.ci-dot[class*="bg-[var(--color-error)]"] {
		box-shadow: 0 0 6px oklch(0.62 0.19 22 / 0.4);
	}

	.detail-panel {
		animation: fade-in 0.2s ease-out;
	}

	.clone-bar {
		animation: slide-up 0.2s ease-out;
	}
</style>

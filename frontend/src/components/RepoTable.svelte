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
	} from "$lib/store";
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

<table class="w-full border-separate border-spacing-0">
	<thead>
		<tr class="border-b border-[var(--color-border)]">
			<th class="w-10 px-4 py-3 text-left">
				<input
					type="checkbox"
					disabled
					class="pointer-events-none opacity-0"
					aria-label="Select all"
				/>
			</th>
			<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-[var(--color-fg-subtle)]">
				Repository
			</th>
			<th class="hidden px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-[var(--color-fg-subtle)] md:table-cell">
				Visibility
			</th>
			<th class="hidden px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-[var(--color-fg-subtle)] lg:table-cell">
				Branch
			</th>
			<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-[var(--color-fg-subtle)]">
				CI
			</th>
			<th class="hidden px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-[var(--color-fg-subtle)] sm:table-cell">
				Complete
			</th>
			<th class="hidden px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-[var(--color-fg-subtle)] sm:table-cell">
				Updated
			</th>
			<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-[var(--color-fg-subtle)]">
				Status
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
				class="group border-b border-[var(--color-border-subtle)] hover:bg-[var(--color-bg-elevated)] {isFocused
					? 'bg-[var(--color-bg-elevated)]'
					: ''} {isSelected
					? 'bg-[var(--color-accent)]/5'
					: ''}"
				onclick={() => focusRow(index)}
				onkeydown={(e) => handleKeydown(e, index)}
				tabindex={isFocused ? 0 : -1}
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
							class="h-4 w-4 rounded border-[var(--color-border)] bg-[var(--color-bg-surface)] accent-[var(--color-accent)]"
							aria-label="Select {repo.Name}"
						/>
					{/if}
				</td>

				<!-- Repo name -->
				<td class="max-w-[200px] px-4 py-3">
					<button
						onclick={() => toggleRepoExpanded(repo.Name)}
						class="flex items-center gap-2 font-medium text-[var(--color-accent)] hover:underline"
					>
						{#if isCloning}
							<span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-[var(--color-fg-muted)] border-t-transparent"></span>
						{/if}
						<span class="truncate">{repo.Name}</span>
					</button>
					{#if repo.Language}
						<span class="text-xs text-[var(--color-fg-subtle)]">{repo.Language}</span>
					{/if}
				</td>

				<!-- Visibility badge -->
				<td class="hidden px-4 py-3 md:table-cell">
					<span
						class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {getVisibilityColor(
							repo.Visibility
						)} bg-current/10"
					>
						{repo.Visibility}
					</span>
				</td>

				<!-- Branch -->
				<td class="hidden px-4 py-3 text-sm text-[var(--color-fg-muted)] lg:table-cell">
					{#if repo.Cloned}
						<div class="flex items-center gap-1">
							<TerminalIcon size="12" />
							<span class="font-mono text-xs">{repo.Branch}</span>
							{#if repo.Dirty}
								<span class="rounded bg-[var(--color-warning)] px-1 text-xs text-[var(--color-bg-base)]">
									dirty
								</span>
							{/if}
						</div>
					{:else}
						<span class="text-[var(--color-fg-subtle)]">—</span>
					{/if}
				</td>

				<!-- CI Status -->
				<td class="px-4 py-3">
					<div
						class="h-2.5 w-2.5 rounded-full {getCIStatusColor(repo.ActionsStatus)}"
						title={repo.ActionsStatus}
					></div>
				</td>

				<!-- Completeness indicators -->
				<td class="hidden px-4 py-3 sm:table-cell">
					{#if issues.length > 0}
						<div class="flex items-center gap-1" title={`Missing: ${issues.join(", ")}`}>
							{#if issues.includes("description")}
								<AlertCircleIcon size="14" class="text-[var(--color-warning)]" />
							{/if}
							{#if issues.includes("README")}
								<FileTextIcon size="14" class="text-[var(--color-warning)]" />
							{/if}
							{#if issues.length > 2}
								<span class="text-xs text-[var(--color-fg-subtle)]">+{issues.length - 2}</span>
							{/if}
						</div>
					{:else}
						<span class="text-[var(--color-success)]">✓</span>
					{/if}
				</td>

				<!-- Last update -->
				<td class="hidden px-4 py-3 text-sm text-[var(--color-fg-muted)] sm:table-cell">
					{formatRelativeTime(repo.GitHubLastPush)}
				</td>

				<!-- Lifecycle badge -->
				<td class="px-4 py-3">
					<span
						class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium uppercase {getLifecycleColor(
							repo.Lifecycle
						)}"
					>
						{repo.Lifecycle}
					</span>
				</td>
			</tr>

			<!-- Expanded row detail -->
			{#if isExpanded}
				<tr class="border-b border-[var(--color-border-subtle)] bg-[var(--color-bg-surface)]">
					<td colspan="8" class="p-4">
						<RepoDetail {repo} />
					</td>
				</tr>
			{/if}
		{/each}

		{#if filteredRepos().length === 0}
			<tr>
				<td colspan="8" class="px-4 py-12 text-center text-[var(--color-fg-muted)]">
					<p class="text-lg">No repositories found</p>
					<p class="mt-1 text-sm">Try adjusting your filters</p>
				</td>
			</tr>
		{/if}
	</tbody>
</table>

<!-- Action bar for cloning -->
{#if selectedRepos.size > 0}
	<div class="fixed bottom-0 left-0 right-0 border-t border-[var(--color-border)] bg-[var(--color-bg-surface)] px-4 py-3 shadow-lg">
		<div class="flex items-center justify-between">
			<div class="text-sm text-[var(--color-fg-base)]">
				{selectedRepos.size} repo{selectedRepos.size === 1 ? "" : "s"} selected
			</div>
			<button
				onclick={cloneSelected}
				disabled={!canClone}
				class="flex items-center gap-2 rounded-md bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-[var(--color-bg-base)] hover:bg-[var(--color-accent-hover)] disabled:opacity-50 disabled:cursor-not-allowed"
			>
				<DownloadIcon size="16" />
				Clone Selected
			</button>
		</div>
	</div>
{/if}

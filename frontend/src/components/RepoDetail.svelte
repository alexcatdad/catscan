<!-- Expanded row detail panel -->
<script lang="ts">
	import {
		GithubIcon,
		GlobeIcon,
		TagIcon,
		GitPullRequestIcon,
		ShieldIcon,
		CheckIcon,
		XIcon
	} from "svelte-feather-icons";
	import { formatRelativeTime } from "$lib/utils";
	import type { Repo } from "$lib/types";

interface Props {
	repo: Repo;
}

const { repo }: Props = $props();

let detailRef = $state<HTMLDivElement>();

const completenessItems = [
	{ key: "HasDescription", label: "Description" },
	{ key: "HasReadme", label: "README" },
	{ key: "HasLicense", label: "License" },
	{ key: "HasTopics", label: "Topics" },
	{ key: "HasPages", label: "GitHub Pages" },
	{ key: "HasHomepage", label: "Homepage" },
	{ key: "HasProjectJson", label: ".project.json" },
	{ key: "HasClaudeMd", label: "CLAUDE.md" },
	{ key: "HasAgentsMd", label: "AGENTS.md" },
];
</script>

<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3" data-testid="repo-detail" bind:this={detailRef}>
	<!-- Description -->
	<div class="detail-card md:col-span-2">
		<h3 class="detail-heading">Description</h3>
		<p class="text-sm text-[var(--color-fg-base)]" data-testid="repo-description">
			{#if repo.Description}
				{repo.Description}
			{:else}
				<span class="italic text-[var(--color-fg-subtle)]">No description</span>
			{/if}
		</p>
	</div>

	<!-- Links -->
	<div class="detail-card">
		<h3 class="detail-heading">Links</h3>
		<div class="flex flex-wrap gap-3">
			<a
				href={`https://github.com/${repo.Name}`}
				target="_blank"
				rel="noopener noreferrer"
				class="flex items-center gap-1.5 rounded-md px-2 py-1 text-sm text-[var(--color-accent)] transition-colors hover:bg-[var(--color-accent)]/10"
			>
				<GithubIcon size="14" />
				GitHub
			</a>
			{#if repo.HomepageURL}
				<a
					href={repo.HomepageURL}
					target="_blank"
					rel="noopener noreferrer"
					class="flex items-center gap-1.5 rounded-md px-2 py-1 text-sm text-[var(--color-accent)] transition-colors hover:bg-[var(--color-accent)]/10"
				>
					<GlobeIcon size="14" />
					Homepage
				</a>
			{/if}
		</div>
	</div>

	<!-- Topics -->
	{#if repo.Topics && repo.Topics.length > 0}
		<div class="detail-card md:col-span-2">
		<h3 class="detail-heading">Topics</h3>
		<div class="flex flex-wrap gap-1.5" data-testid="repo-topics">
			{#each repo.Topics as topic}
				<span
					class="rounded-full border border-[var(--color-border-subtle)] bg-[var(--color-bg-elevated)]/60 px-2.5 py-0.5 font-[var(--font-mono)] text-[11px] text-[var(--color-fg-muted)]"
				>
					{topic}
				</span>
			{/each}
		</div>
	</div>
	{/if}

	<!-- Latest Release -->
	<div class="detail-card">
		<h3 class="detail-heading">Latest Release</h3>
		{#if repo.LatestRelease}
			<div class="flex items-center gap-2">
				<TagIcon size="14" class="text-[var(--color-accent)]" />
				<span class="font-[var(--font-mono)] text-sm text-[var(--color-fg-base)]">{repo.LatestRelease.TagName}</span>
				<span class="font-[var(--font-mono)] text-xs text-[var(--color-fg-subtle)]">
					{formatRelativeTime(repo.LatestRelease.PublishedAt)}
				</span>
			</div>
		{:else}
			<span class="text-sm text-[var(--color-fg-subtle)]">No releases</span>
		{/if}
	</div>

	<!-- Pull Requests -->
	<div class="detail-card">
		<h3 class="detail-heading">Open Pull Requests</h3>
		<div class="flex items-center gap-2">
			<GitPullRequestIcon size="14" class="text-[var(--color-accent)]" />
			<span class="font-[var(--font-mono)] text-sm text-[var(--color-fg-base)]">{repo.OpenPRs} open</span>
		</div>
	</div>

	<!-- Completeness Checklist -->
	<div class="detail-card md:col-span-2">
		<h3 class="detail-heading">Completeness</h3>
		<div class="grid grid-cols-3 gap-x-4 gap-y-2 sm:grid-cols-4 md:grid-cols-5" data-testid="repo-completeness">
			{#each completenessItems as item}
				<div class="flex items-center gap-2 text-sm">
					{#if repo.Completeness[item.key]}
						<span class="inline-flex h-4 w-4 items-center justify-center rounded-full bg-[var(--color-success)]/15">
							<CheckIcon size="10" class="text-[var(--color-success)]" />
						</span>
						<span class="text-[var(--color-fg-base)]">{item.label}</span>
					{:else}
						<span class="inline-flex h-4 w-4 items-center justify-center rounded-full bg-[var(--color-fg-subtle)]/10">
							<XIcon size="10" class="text-[var(--color-fg-subtle)]" />
						</span>
						<span class="text-[var(--color-fg-subtle)]">{item.label}</span>
					{/if}
				</div>
			{/each}
		</div>
	</div>

	<!-- Branch Protection -->
	<div class="detail-card">
		<h3 class="detail-heading">Branch Protection</h3>
		<div class="flex items-center gap-2">
			<ShieldIcon size="14" class="text-[var(--color-fg-subtle)]" />
			<span class="text-sm text-[var(--color-fg-muted)]">Protected branches</span>
		</div>
	</div>
</div>

<style>
	.detail-card {
		padding: 12px 16px;
		border-radius: 10px;
		border: 1px solid oklch(0.19 0.012 265);
		background: oklch(0.135 0.017 265 / 0.6);
		backdrop-filter: blur(8px);
	}

	.detail-heading {
		margin-bottom: 8px;
		font-family: "Azeret Mono", ui-monospace, monospace;
		font-size: 10px;
		font-weight: 500;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: oklch(0.55 0.018 260);
	}
</style>

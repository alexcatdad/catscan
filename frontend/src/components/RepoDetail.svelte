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
	<div class="md:col-span-2">
		<h3 class="mb-2 text-sm font-medium text-[var(--color-fg-subtle)]">Description</h3>
		<p class="text-sm text-[var(--color-fg-base)]" data-testid="repo-description">
			{#if repo.Description}
				{repo.Description}
			{:else}
				<span class="text-[var(--color-fg-subtle)] italic">No description</span>
			{/if}
		</p>
	</div>

	<!-- Links -->
	<div>
		<h3 class="mb-2 text-sm font-medium text-[var(--color-fg-subtle)]">Links</h3>
		<div class="flex flex-wrap gap-2">
			<a
				href={`https://github.com/${repo.Name}`}
				target="_blank"
				rel="noopener noreferrer"
				class="flex items-center gap-1 text-sm text-[var(--color-accent)] hover:underline"
			>
				<GithubIcon size="14" />
				GitHub
			</a>
			{#if repo.HomepageURL}
				<a
					href={repo.HomepageURL}
					target="_blank"
					rel="noopener noreferrer"
					class="flex items-center gap-1 text-sm text-[var(--color-accent)] hover:underline"
				>
					<GlobeIcon size="14" />
					Homepage
				</a>
			{/if}
		</div>
	</div>

	<!-- Topics -->
	{#if repo.Topics && repo.Topics.length > 0}
		<div class="md:col-span-2">
		<h3 class="mb-2 text-sm font-medium text-[var(--color-fg-subtle)]">Topics</h3>
		<div class="flex flex-wrap gap-1" data-testid="repo-topics">
			{#each repo.Topics as topic}
				<span
					class="rounded-full bg-[var(--color-bg-elevated)] px-2 py-0.5 text-xs text-[var(--color-fg-base)]"
				>
					{topic}
				</span>
			{/each}
		</div>
	</div>
	{/if}

	<!-- Latest Release -->
	<div>
		<h3 class="mb-2 text-sm font-medium text-[var(--color-fg-subtle)]">Latest Release</h3>
		{#if repo.LatestRelease}
			<div class="flex items-center gap-2">
				<TagIcon size="14" class="text-[var(--color-accent)]" />
				<span class="text-sm text-[var(--color-fg-base)]">{repo.LatestRelease.TagName}</span>
				<span class="text-xs text-[var(--color-fg-muted)]">
					{formatRelativeTime(repo.LatestRelease.PublishedAt)}
				</span>
			</div>
		{:else}
			<span class="text-sm text-[var(--color-fg-subtle)]">No releases</span>
		{/if}
	</div>

	<!-- Pull Requests -->
	<div>
		<h3 class="mb-2 text-sm font-medium text-[var(--color-fg-subtle)]">Open Pull Requests</h3>
		<div class="flex items-center gap-2">
			<GitPullRequestIcon size="14" class="text-[var(--color-accent)]" />
			<span class="text-sm text-[var(--color-fg-base)]">{repo.OpenPRs} open</span>
		</div>
	</div>

	<!-- Completeness Checklist -->
	<div class="md:col-span-2">
		<h3 class="mb-2 text-sm font-medium text-[var(--color-fg-subtle)]">Completeness</h3>
		<div class="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-5" data-testid="repo-completeness">
			{#each completenessItems as item}
				<div class="flex items-center gap-2 text-sm">
					{#if repo.Completeness[item.key]}
						<CheckIcon size="14" class="text-[var(--color-success)]" />
						<span class="text-[var(--color-fg-base)]">{item.label}</span>
					{:else}
						<XIcon size="14" class="text-[var(--color-fg-subtle)]" />
						<span class="text-[var(--color-fg-subtle)] line-through">{item.label}</span>
					{/if}
				</div>
			{/each}
		</div>
	</div>

	<!-- Branch Protection -->
	<div>
		<h3 class="mb-2 text-sm font-medium text-[var(--color-fg-subtle)]">Branch Protection</h3>
		<div class="flex items-center gap-2">
			<ShieldIcon size="14" />
			<span class="text-sm text-[var(--color-fg-base)]">Protected branches</span>
		</div>
	</div>
</div>

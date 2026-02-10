<!-- Summary stat cards that act as filters -->
<script lang="ts">
import type { FilterOptions } from "$lib/types";
import { clearFilters, filters, setFilters, summaryStats } from "$lib/store.svelte";

interface StatCard {
	key: string;
	label: string;
	count: number;
	filterKey?: keyof FilterOptions;
	filterValue?: string | boolean;
}

const statsCards = $derived.by(() => {
	const stats = summaryStats();
	const cards: StatCard[] = [
		{ key: "total", label: "Total", count: stats.total },
		{ key: "cloned", label: "Cloned", count: stats.cloned, filterKey: "cloned", filterValue: true },
		{
			key: "public",
			label: "Public",
			count: stats.public,
			filterKey: "visibility",
			filterValue: "public",
		},
		{
			key: "private",
			label: "Private",
			count: stats.private,
			filterKey: "visibility",
			filterValue: "private",
		},
		{
			key: "ongoing",
			label: "Ongoing",
			count: stats.ongoing,
			filterKey: "lifecycle",
			filterValue: "ongoing",
		},
		{
			key: "maintenance",
			label: "Maintenance",
			count: stats.maintenance,
			filterKey: "lifecycle",
			filterValue: "maintenance",
		},
		{
			key: "stale",
			label: "Stale",
			count: stats.stale,
			filterKey: "lifecycle",
			filterValue: "stale",
		},
		{
			key: "abandoned",
			label: "Abandoned",
			count: stats.abandoned,
			filterKey: "lifecycle",
			filterValue: "abandoned",
		},
	];
	return cards;
});

function isActive(card: StatCard): boolean {
	if (!card.filterKey) return false;
	return filters()[card.filterKey] === card.filterValue;
}

function applyFilter(card: StatCard): void {
	if (!card.filterKey) return;

	const currentFilters = filters();
	const currentValue = currentFilters[card.filterKey];
	if (currentValue === card.filterValue) {
		// Toggle off - clear this filter
		const { [card.filterKey]: _, ...rest } = currentFilters;
		setFilters(rest);
	} else {
		// Apply this filter
		setFilters({ ...currentFilters, [card.filterKey]: card.filterValue });
	}
}

function clearAllFilters(): void {
	clearFilters();
}
</script>

<div class="mb-6 flex flex-wrap items-center gap-2" data-testid="summary-cards">
	{#each statsCards as card}
		<button
			onclick={() => applyFilter(card)}
			class="stat-card group relative flex items-center gap-3 rounded-lg border px-4 py-2.5 transition-all duration-200
				{isActive(card)
					? 'border-[var(--color-accent)]/50 bg-[var(--color-accent)]/8 shadow-[0_0_16px_oklch(0.72_0.14_192/0.1)]'
					: 'border-[var(--color-border-subtle)] bg-[var(--color-bg-surface)]/60 hover:border-[var(--color-border)] hover:bg-[var(--color-bg-surface)]'}"
		>
			<span class="font-[var(--font-mono)] text-xs uppercase tracking-wider text-[var(--color-fg-subtle)] transition-colors group-hover:text-[var(--color-fg-muted)]
				{isActive(card) ? '!text-[var(--color-accent)]' : ''}">{card.label}</span>
			<span class="font-[var(--font-mono)] text-lg font-medium tabular-nums text-[var(--color-fg-base)]
				{isActive(card) ? '!text-[var(--color-accent)]' : ''}">{card.count}</span>
		</button>
	{/each}

	{#if Object.keys(filters()).length > 0}
		<button
			onclick={clearAllFilters}
			class="rounded-lg border border-[var(--color-border-subtle)] px-3 py-2 font-[var(--font-mono)] text-xs text-[var(--color-fg-subtle)] transition-all hover:border-[var(--color-error)]/30 hover:text-[var(--color-error)]"
		>
			Clear
		</button>
	{/if}
</div>

<!-- Summary stat cards that act as filters -->
<script lang="ts">
import { clearFilters, filters, setFilters, summaryStats } from "$lib/store";

interface StatCard {
	key: string;
	label: string;
	count: number;
	filterKey?: keyof typeof filters;
	filterValue?: string | boolean;
}

const statsCards = $derived(() => {
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
	return filters[card.filterKey] === card.filterValue;
}

function applyFilter(card: StatCard): void {
	if (!card.filterKey) return;

	const currentValue = filters[card.filterKey];
	if (currentValue === card.filterValue) {
		// Toggle off - clear this filter
		const { [card.filterKey]: _, ...rest } = filters;
		setFilters(rest);
	} else {
		// Apply this filter
		setFilters({ ...filters, [card.filterKey]: card.filterValue });
	}
}

function clearAllFilters(): void {
	clearFilters();
}
</script>

<div class="mb-4 flex flex-wrap items-center gap-3">
	{#each statsCards() as card}
		<button
			onclick={() => applyFilter(card)}
			class="group relative flex min-w-[100px] flex-1 items-center justify-between rounded-md border border-[var(--color-border)] bg-[var(--color-bg-surface)] px-4 py-2 text-left transition-colors hover:border-[var(--color-accent)] {isActive(card)
				? 'border-[var(--color-accent)] bg-[var(--color-accent)]/10'
				: ''}"
		>
			<span class="text-sm text-[var(--color-fg-muted)] group-hover:text-[var(--color-fg-base)]">{card.label}</span>
			<span class="text-lg font-semibold text-[var(--color-fg-base)]">{card.count}</span>

			{#if isActive(card)}
				<span class="absolute right-1 top-1 h-2 w-2 rounded-full bg-[var(--color-accent)]"></span>
			{/if}
		</button>
	{/each}

	{#if Object.keys(filters).length > 0}
		<button
			onclick={clearAllFilters}
			class="rounded-md border border-[var(--color-border)] px-3 py-2 text-sm text-[var(--color-fg-muted)] hover:bg-[var(--color-bg-elevated)] hover:text-[var(--color-fg-base)]"
		>
			Clear All
		</button>
	{/if}
</div>

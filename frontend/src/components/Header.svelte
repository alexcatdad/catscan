<!-- Header bar with logo, filters, sort controls, and settings button -->
<script lang="ts">
	import { SettingsIcon, FilterIcon, XIcon } from "svelte-feather-icons";
	import type { FilterOptions, Lifecycle, SortOptions, Visibility } from "$lib/types";
	import {
		setFilters,
		clearFilters,
		setSort,
		sort,
		filters,
	} from "$lib/store";

	const lifecycleOptions: { value: Lifecycle; label: string }[] = [
		{ value: "ongoing", label: "Ongoing" },
		{ value: "maintenance", label: "Maintenance" },
		{ value: "stale", label: "Stale" },
		{ value: "abandoned", label: "Abandoned" },
	];

	const visibilityOptions: { value: Visibility; label: string }[] = [
		{ value: "public", label: "Public" },
		{ value: "private", label: "Private" },
	];

	const sortFieldOptions: { value: SortOptions["field"]; label: string }[] = [
		{ value: "name", label: "Name" },
		{ value: "lastUpdate", label: "Last Update" },
		{ value: "lifecycle", label: "Lifecycle" },
	];

	let showFilterMenu = $state(false);
	let showSortMenu = $state(false);
	let showSettings = $state(false);

	let filterMenuRef = $state<HTMLDivElement>();
	let sortMenuRef = $state<HTMLDivElement>();

	function handleClickOutside(event: MouseEvent) {
		if (filterMenuRef && !filterMenuRef.contains(event.target as Node)) {
			showFilterMenu = false;
		}
		if (sortMenuRef && !sortMenuRef.contains(event.target as Node)) {
			showSortMenu = false;
		}
	}

	function setLifecycleFilter(value: string) {
		if (value === "") {
			const { lifecycle, ...rest } = filters;
			setFilters(rest);
		} else {
			setFilters({ ...filters, lifecycle: value });
		}
		showFilterMenu = false;
	}

	function setVisibilityFilter(value: string) {
		if (value === "") {
			const { visibility, ...rest } = filters;
			setFilters(rest);
		} else {
			setFilters({ ...filters, visibility: value });
		}
		showFilterMenu = false;
	}

	function setClonedFilter(value: boolean | undefined) {
		if (value === undefined) {
			const { cloned, ...rest } = filters;
			setFilters(rest);
		} else {
			setFilters({ ...filters, cloned: value });
		}
		showFilterMenu = false;
	}

	function setSortField(field: SortOptions["field"]) {
		setSort({ ...sort, field });
		showSortMenu = false;
	}

	function toggleSortOrder() {
		setSort({ ...sort, order: sort.order === "asc" ? "desc" : "asc" });
	}

	function hasActiveFilters(): boolean {
		return Object.keys(filters).length > 0;
	}

	const activeFilterCount = $derived(
		Object.keys(filters).filter((k) => filters[k as keyof FilterOptions] !== undefined).length
	);
</script>

<svelte:window onclick={handleClickOutside} />

<header class="border-b border-[var(--color-border)] bg-[var(--color-bg-surface)] px-4 py-3">
	<div class="flex items-center justify-between gap-4">
		<!-- Logo and name -->
		<div class="flex items-center gap-3">
			<div class="flex h-8 w-8 items-center justify-center rounded bg-[var(--color-accent)] text-[var(--color-bg-base)] font-bold">
				C
			</div>
			<h1 class="text-lg font-semibold text-[var(--color-fg-base)]">CatScan</h1>
		</div>

		<!-- Filters and sort -->
		<div class="flex items-center gap-2">
		<!-- Filter button -->
		<div class="relative">
			<button
				onclick={() => (showFilterMenu = !showFilterMenu)}
				class="flex items-center gap-2 rounded-md border border-[var(--color-border)] px-3 py-1.5 text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-elevated)]"
			>
				<FilterIcon size="14" />
				<span>Filters</span>
				{#if activeFilterCount > 0}
					<span
						class="flex h-5 w-5 items-center justify-center rounded-full bg-[var(--color-accent)] text-xs text-[var(--color-bg-base)]"
					>
						{activeFilterCount}
					</span>
				{/if}
			</button>

			{#if showFilterMenu}
				<div
					bind:this={filterMenuRef}
					class="absolute right-0 top-full z-50 mt-1 w-48 rounded-md border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-2 shadow-lg"
					role="menu"
				>
					<!-- Lifecycle filter -->
					<div class="mb-2">
						<div class="mb-1 px-2 text-xs text-[var(--color-fg-subtle)]">Lifecycle</div>
						<button
							onclick={() => setLifecycleFilter("")}
							class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
							role="menuitem"
						>
							All
						</button>
						{#each lifecycleOptions as option}
							<button
								onclick={() => setLifecycleFilter(option.value)}
								class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
								role="menuitem"
							>
								{option.label}
							</button>
						{/each}
					</div>

					<!-- Visibility filter -->
					<div class="mb-2">
						<div class="mb-1 px-2 text-xs text-[var(--color-fg-subtle)]">Visibility</div>
						<button
							onclick={() => setVisibilityFilter("")}
							class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
							role="menuitem"
						>
							All
						</button>
						{#each visibilityOptions as option}
							<button
								onclick={() => setVisibilityFilter(option.value)}
								class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
								role="menuitem"
							>
								{option.label}
							</button>
						{/each}
					</div>

					<!-- Cloned filter -->
					<div>
						<div class="mb-1 px-2 text-xs text-[var(--color-fg-subtle)]">Clone Status</div>
						<button
							onclick={() => setClonedFilter(undefined)}
							class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
							role="menuitem"
						>
							All
						</button>
						<button
							onclick={() => setClonedFilter(true)}
							class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
							role="menuitem"
						>
							Cloned
						</button>
						<button
							onclick={() => setClonedFilter(false)}
							class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
							role="menuitem"
						>
							Not Cloned
						</button>
					</div>

					{#if hasActiveFilters()}
						<div class="mt-2 border-t border-[var(--color-border-subtle)] pt-2">
							<button
								onclick={clearFilters}
								class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-error)] hover:bg-[var(--color-bg-base)]"
								role="menuitem"
							>
								Clear Filters
							</button>
						</div>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Sort button -->
		<div class="relative">
			<button
				onclick={() => (showSortMenu = !showSortMenu)}
				class="flex items-center gap-2 rounded-md border border-[var(--color-border)] px-3 py-1.5 text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-elevated)]"
			>
				Sort: {sortFieldOptions.find((o) => o.value === sort.field)?.label}
				{#if sort.order === "asc"}
					↑
				{:else}
					↓
				{/if}
			</button>

			{#if showSortMenu}
				<div
					bind:this={sortMenuRef}
					class="absolute right-0 top-full z-50 mt-1 w-40 rounded-md border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-2 shadow-lg"
					role="menu"
				>
					{#each sortFieldOptions as option}
						<button
							onclick={() => setSortField(option.value)}
							class="block w-full rounded px-2 py-1 text-left text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
							role="menuitem"
						>
							{option.label}
						</button>
					{/each}
					<div class="my-1 border-t border-[var(--color-border-subtle)]"></div>
					<button
						onclick={toggleSortOrder}
						class="flex items-center justify-between rounded px-2 py-1 text-sm text-[var(--color-fg-base)] hover:bg-[var(--color-bg-base)]"
						role="menuitem"
					>
						<span>Reverse</span>
						<span class="text-[var(--color-fg-muted)]">{sort.order === "asc" ? "↓" : "↑"}</span>
					</button>
				</div>
			{/if}
		</div>

		<!-- Settings button -->
		<button
			onclick={() => (showSettings = !showSettings)}
			class="rounded-md p-2 text-[var(--color-fg-muted)] hover:bg-[var(--color-bg-elevated)] hover:text-[var(--color-fg-base)]"
			aria-label="Settings"
		>
			<SettingsIcon size="18" />
		</button>
		</div>
	</div>
</header>

{#if showSettings}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onclick={() => (showSettings = false)}>
		<div class="w-full max-w-md rounded-md border border-[var(--color-border)] bg-[var(--color-bg-surface)] p-6 shadow-lg" onclick={(e) => e.stopPropagation()}>
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold text-[var(--color-fg-base)]">Settings</h2>
				<button onclick={() => (showSettings = false)} class="text-[var(--color-fg-muted)] hover:text-[var(--color-fg-base)]">
					<XIcon size="20" />
				</button>
			</div>
			<p class="text-sm text-[var(--color-fg-muted)]">
				Settings panel coming soon. For now, edit the config file directly.
			</p>
		</div>
	</div>
{/if}

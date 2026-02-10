<!-- Header bar with logo, filters, sort controls, and settings button -->
<script lang="ts">
	import { SettingsIcon, FilterIcon, XIcon } from "svelte-feather-icons";
	import SettingsPanel from "./SettingsPanel.svelte";
	import type { FilterOptions, Lifecycle, SortOptions, Visibility } from "$lib/types";
	import {
		setFilters,
		clearFilters,
		setSort,
		sort,
		filters,
	} from "$lib/store.svelte";

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
			const { lifecycle, ...rest } = filters();
			setFilters(rest);
		} else {
			setFilters({ ...filters(), lifecycle: value });
		}
		showFilterMenu = false;
	}

	function setVisibilityFilter(value: string) {
		if (value === "") {
			const { visibility, ...rest } = filters();
			setFilters(rest);
		} else {
			setFilters({ ...filters(), visibility: value });
		}
		showFilterMenu = false;
	}

	function setClonedFilter(value: boolean | undefined) {
		if (value === undefined) {
			const { cloned, ...rest } = filters();
			setFilters(rest);
		} else {
			setFilters({ ...filters(), cloned: value });
		}
		showFilterMenu = false;
	}

	function setSortField(field: SortOptions["field"]) {
		setSort({ ...sort(), field });
		showSortMenu = false;
	}

	function toggleSortOrder() {
		setSort({ ...sort(), order: sort().order === "asc" ? "desc" : "asc" });
	}

	function hasActiveFilters(): boolean {
		return Object.keys(filters()).length > 0;
	}

	const activeFilterCount = $derived(
		Object.keys(filters()).filter((k) => filters()[k as keyof FilterOptions] !== undefined).length
	);
</script>

<svelte:window onclick={handleClickOutside} />

<header class="header-bar relative overflow-hidden border-b border-[var(--color-border)] bg-[var(--color-bg-surface)]">
	<div class="relative z-10 flex items-center justify-between gap-4 px-6 py-3">
		<!-- Logo and name -->
		<div class="flex items-center gap-3">
			<div class="logo-mark flex h-8 w-8 items-center justify-center rounded-lg font-[var(--font-mono)] text-sm font-medium text-[var(--color-bg-base)]">
				CS
			</div>
			<h1 class="font-[var(--font-mono)] text-base font-medium tracking-tight text-[var(--color-fg-base)]">
				CatScan
			</h1>
		</div>

		<!-- Controls -->
		<div class="flex items-center gap-2">
			<!-- Filter button -->
			<div class="relative">
				<button
					onclick={() => (showFilterMenu = !showFilterMenu)}
					class="flex items-center gap-2 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-sm text-[var(--color-fg-muted)] transition-all hover:border-[var(--color-fg-subtle)] hover:text-[var(--color-fg-base)]"
				>
					<FilterIcon size="14" />
					<span>Filters</span>
					{#if activeFilterCount > 0}
						<span
							class="flex h-4 w-4 items-center justify-center rounded-full bg-[var(--color-accent)] text-[10px] font-medium text-[var(--color-bg-base)]"
						>
							{activeFilterCount}
						</span>
					{/if}
				</button>

				{#if showFilterMenu}
					<div
						bind:this={filterMenuRef}
						class="dropdown-menu absolute right-0 top-full z-50 mt-2 w-52 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-2 shadow-xl shadow-black/30"
						role="menu"
					>
						<!-- Lifecycle filter -->
						<div class="mb-2">
							<div class="mb-1 px-2 py-1 font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Lifecycle</div>
							<button
								onclick={() => setLifecycleFilter("")}
								class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
								role="menuitem"
							>
								All
							</button>
							{#each lifecycleOptions as option}
								<button
									onclick={() => setLifecycleFilter(option.value)}
									class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
									role="menuitem"
								>
									{option.label}
								</button>
							{/each}
						</div>

						<!-- Visibility filter -->
						<div class="mb-2 border-t border-[var(--color-border-subtle)] pt-2">
							<div class="mb-1 px-2 py-1 font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Visibility</div>
							<button
								onclick={() => setVisibilityFilter("")}
								class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
								role="menuitem"
							>
								All
							</button>
							{#each visibilityOptions as option}
								<button
									onclick={() => setVisibilityFilter(option.value)}
									class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
									role="menuitem"
								>
									{option.label}
								</button>
							{/each}
						</div>

						<!-- Cloned filter -->
						<div class="border-t border-[var(--color-border-subtle)] pt-2">
							<div class="mb-1 px-2 py-1 font-[var(--font-mono)] text-[10px] uppercase tracking-widest text-[var(--color-fg-subtle)]">Clone Status</div>
							<button
								onclick={() => setClonedFilter(undefined)}
								class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
								role="menuitem"
							>
								All
							</button>
							<button
								onclick={() => setClonedFilter(true)}
								class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
								role="menuitem"
							>
								Cloned
							</button>
							<button
								onclick={() => setClonedFilter(false)}
								class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
								role="menuitem"
							>
								Not Cloned
							</button>
						</div>

						{#if hasActiveFilters()}
							<div class="mt-2 border-t border-[var(--color-border-subtle)] pt-2">
								<button
									onclick={clearFilters}
									class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-error)] transition-colors hover:bg-[var(--color-error)]/10"
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
					class="flex items-center gap-2 rounded-lg border border-[var(--color-border)] px-3 py-1.5 font-[var(--font-mono)] text-xs text-[var(--color-fg-muted)] transition-all hover:border-[var(--color-fg-subtle)] hover:text-[var(--color-fg-base)]"
				>
					{sortFieldOptions.find((o) => o.value === sort().field)?.label}
					<span class="text-[var(--color-accent)]">
						{#if sort().order === "asc"}↑{:else}↓{/if}
					</span>
				</button>

				{#if showSortMenu}
					<div
						bind:this={sortMenuRef}
						class="dropdown-menu absolute right-0 top-full z-50 mt-2 w-44 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-2 shadow-xl shadow-black/30"
						role="menu"
					>
						{#each sortFieldOptions as option}
							<button
								onclick={() => setSortField(option.value)}
								class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
								role="menuitem"
							>
								{option.label}
							</button>
						{/each}
						<div class="my-1 border-t border-[var(--color-border-subtle)]"></div>
						<button
							onclick={toggleSortOrder}
							class="flex w-full items-center justify-between rounded-lg px-2 py-1.5 text-sm text-[var(--color-fg-base)] transition-colors hover:bg-[var(--color-bg-surface)]"
							role="menuitem"
						>
							<span>Reverse</span>
							<span class="font-[var(--font-mono)] text-[var(--color-accent)]">{sort().order === "asc" ? "↓" : "↑"}</span>
						</button>
					</div>
				{/if}
			</div>

			<!-- Settings button -->
			<button
				onclick={() => (showSettings = !showSettings)}
				class="rounded-lg p-2 text-[var(--color-fg-subtle)] transition-all hover:bg-[var(--color-bg-elevated)] hover:text-[var(--color-fg-base)]"
				aria-label="Settings"
				data-testid="settings-button"
			>
				<SettingsIcon size="16" />
			</button>
		</div>
	</div>

	<!-- Scan line effect -->
	<div class="scan-line"></div>
</header>

<SettingsPanel show={showSettings} onClose={() => (showSettings = false)} />

<style>
	.logo-mark {
		background: linear-gradient(
			135deg,
			oklch(0.72 0.14 192),
			oklch(0.62 0.16 200)
		);
		box-shadow: 0 0 16px oklch(0.72 0.14 192 / 0.25);
	}

	.header-bar {
		background: linear-gradient(
			180deg,
			oklch(0.155 0.02 265) 0%,
			oklch(0.145 0.018 265) 100%
		);
	}

	.scan-line {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 1px;
		background: linear-gradient(
			90deg,
			transparent 0%,
			oklch(0.72 0.14 192 / 0.4) 50%,
			transparent 100%
		);
		animation: scanline 6s ease-in-out infinite;
		pointer-events: none;
	}

	.dropdown-menu {
		animation: fade-in 0.15s ease-out;
		backdrop-filter: blur(12px);
	}
</style>

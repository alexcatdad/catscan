<script lang="ts">
	import { onMount } from "svelte";
	import Header from "./components/Header.svelte";
	import ErrorBanner from "./components/ErrorBanner.svelte";
	import SummaryCards from "./components/SummaryCards.svelte";
	import RepoTable from "./components/RepoTable.svelte";
	import { initializeStore, loading, error } from "$lib/store.svelte";

	onMount(() => {
		initializeStore();
	});
</script>

<div class="min-h-screen bg-[var(--color-bg-base)] text-[var(--color-fg-base)] font-[var(--font-sans)]">
	<Header />
	<ErrorBanner />

	<main class="mx-auto max-w-7xl px-4 py-6">
		{#if loading()}
			<div class="flex items-center justify-center py-20">
				<div class="h-8 w-8 animate-spin rounded-full border-2 border-[var(--color-fg-muted)] border-t-transparent"></div>
				<span class="ml-3 text-[var(--color-fg-muted)]">Loading repositories...</span>
			</div>
		{:else if error()}
			<div class="rounded-md border border-[var(--color-error)] bg-[var(--color-error)]/10 px-4 py-8 text-center">
				<p class="text-[var(--color-error)]">{error()}</p>
				<p class="mt-2 text-sm text-[var(--color-fg-muted)]">Make sure the CatScan server is running.</p>
			</div>
		{:else}
			<SummaryCards />
			<RepoTable />
		{/if}
	</main>
</div>

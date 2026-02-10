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

<div class="app-shell min-h-screen bg-[var(--color-bg-base)] text-[var(--color-fg-base)] font-[var(--font-sans)]">
	<Header />
	<ErrorBanner />

	<main class="mx-auto max-w-[1400px] px-6 py-8">
		{#if loading()}
			<div class="flex flex-col items-center justify-center py-24">
				<div class="scanner-loader mb-4"></div>
				<span class="font-[var(--font-mono)] text-sm tracking-wider text-[var(--color-fg-muted)]">Scanning repositories...</span>
			</div>
		{:else if error()}
			<div class="mx-auto max-w-md rounded-xl border border-[var(--color-error)]/30 bg-[var(--color-error)]/5 px-6 py-10 text-center">
				<p class="font-medium text-[var(--color-error)]">{error()}</p>
				<p class="mt-3 text-sm text-[var(--color-fg-muted)]">Make sure the CatScan server is running.</p>
			</div>
		{:else}
			<SummaryCards />
			<RepoTable />
		{/if}
	</main>
</div>

<style>
	.app-shell {
		background-image:
			radial-gradient(oklch(0.17 0.02 265 / 0.5) 1px, transparent 1px);
		background-size: 32px 32px;
		background-position: 0 0;
	}

	.scanner-loader {
		width: 48px;
		height: 48px;
		border-radius: 50%;
		border: 2px solid oklch(0.24 0.015 265);
		border-top-color: oklch(0.72 0.14 192);
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>

<!-- Error banner for GitHub CLI issues -->
<script lang="ts">
	import { AlertTriangleIcon, XIcon } from "svelte-feather-icons";
	import { ghError, dismissGHError } from "$lib/store.svelte";
</script>

{#if ghError()}
	<div class="error-banner border-b border-[var(--color-error)]/20 bg-[var(--color-error)]/5 px-6 py-3">
		<div class="mx-auto flex max-w-[1400px] items-start gap-3">
			<div class="mt-0.5 text-[var(--color-error)]">
				<AlertTriangleIcon size="16" />
			</div>
			<div class="flex-1">
				<p class="font-[var(--font-mono)] text-xs font-medium text-[var(--color-error)]">GitHub CLI Error</p>
				<p class="mt-1 text-sm text-[var(--color-fg-muted)]">
					{ghError()?.message}
				</p>
			</div>
			<button
				onclick={dismissGHError}
				class="rounded-lg p-1 text-[var(--color-fg-subtle)] transition-colors hover:bg-[var(--color-bg-elevated)] hover:text-[var(--color-fg-base)]"
				aria-label="Dismiss"
			>
				<XIcon size="16" />
			</button>
		</div>
	</div>
{/if}

<style>
	.error-banner {
		animation: fade-in 0.2s ease-out;
		border-left: 3px solid oklch(0.62 0.19 22);
	}
</style>

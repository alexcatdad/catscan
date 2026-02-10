<!-- Error banner for GitHub CLI issues -->
<script lang="ts">
	import { AlertTriangleIcon, XIcon } from "svelte-feather-icons";
	import { ghError, dismissGHError } from "$lib/store";
</script>

{#if ghError}
	<div class="border-b border-[var(--color-error)] bg-[var(--color-error)]/10 px-4 py-3">
		<div class="flex items-start gap-3">
			<div class="mt-0.5 text-[var(--color-error)]">
				<AlertTriangleIcon size="18" />
			</div>
			<div class="flex-1">
				<p class="font-medium text-[var(--color-error)]">GitHub CLI Error</p>
				<p class="mt-1 text-sm text-[var(--color-fg-base)]">
					{ghError.message}
				</p>
			</div>
			<button
				onclick={dismissGHError}
				class="text-[var(--color-fg-muted)] hover:text-[var(--color-fg-base)]"
				aria-label="Dismiss"
			>
				<XIcon size="18" />
			</button>
		</div>
	</div>
{/if}

<script lang="ts">
	// Bottom-Sheet auf dem Handy, zentrierter Dialog auf dem Desktop.
	import type { Snippet } from 'svelte';
	import { X } from '@lucide/svelte';

	let {
		open = $bindable(false),
		title,
		children,
		footer
	}: { open: boolean; title: string; children: Snippet; footer?: Snippet } = $props();

	let dialog: HTMLDialogElement | undefined = $state();

	$effect(() => {
		if (!dialog) return;
		if (open && !dialog.open) dialog.showModal();
		if (!open && dialog.open) dialog.close();
	});
</script>

<dialog
	bind:this={dialog}
	onclose={() => (open = false)}
	onclick={(e) => {
		if (e.target === dialog) open = false;
	}}
	class="sheet m-0 mt-auto max-h-[92dvh] w-full max-w-none overflow-hidden rounded-t-3xl border border-line bg-surface p-0 text-fg shadow-2xl backdrop:bg-black/50 backdrop:backdrop-blur-[2px] md:m-auto md:max-w-lg md:rounded-3xl"
>
	{#if open}
		<div class="flex max-h-[92dvh] flex-col">
			<header class="flex items-center gap-2 border-b border-line px-4 py-2">
				<h2 class="flex-1 truncate text-lg font-bold">{title}</h2>
				<button class="icon-btn" onclick={() => (open = false)} aria-label="Schließen"><X class="size-6" /></button>
			</header>
			<div class="flex-1 overflow-y-auto overscroll-contain px-4 py-4">
				{@render children()}
			</div>
			{#if footer}
				<footer class="border-t border-line px-4 pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))]">
					{@render footer()}
				</footer>
			{/if}
		</div>
	{/if}
</dialog>

<style>
	.sheet[open] {
		animation: sheet-in 180ms ease-out;
	}
	@keyframes sheet-in {
		from {
			transform: translateY(24px);
			opacity: 0;
		}
	}
</style>

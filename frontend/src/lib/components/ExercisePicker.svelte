<script lang="ts">
	import Sheet from './Sheet.svelte';
	import { app } from '#lib/state.svelte.ts';
	import { categoryLabel } from '#lib/categories.ts';
	import type { Exercise } from '#lib/types.ts';
	import { Search, ScanEye } from '@lucide/svelte';

	let { open = $bindable(false), onpick }: { open: boolean; onpick: (e: Exercise) => void } = $props();

	let query = $state('');
	const filtered = $derived(
		app.exercises.filter((e) => !query || e.name.toLowerCase().includes(query.toLowerCase()) || categoryLabel(e.category).toLowerCase().includes(query.toLowerCase()))
	);
	const groups = $derived.by(() => {
		const m = new Map<string, Exercise[]>();
		for (const e of filtered) {
			const k = categoryLabel(e.category);
			if (!m.has(k)) m.set(k, []);
			m.get(k)!.push(e);
		}
		return [...m.entries()].sort((a, b) => a[0].localeCompare(b[0], 'de'));
	});
</script>

<Sheet bind:open title="Übung wählen">
	<div class="relative mb-3">
		<Search class="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-muted" />
		<input class="input pl-10" placeholder="Suchen …" bind:value={query} type="search" />
	</div>
	{#each groups as [cat, list] (cat)}
		<h3 class="mt-4 mb-1 px-1 text-xs font-bold tracking-wider text-muted uppercase">{cat}</h3>
		<div class="divide-y divide-line overflow-hidden rounded-xl border border-line">
			{#each list as e (e.id)}
				<button
					type="button"
					class="flex min-h-14 w-full items-center gap-3 px-3 text-left hover:bg-surface-2 active:bg-surface-2"
					onclick={() => {
						onpick(e);
						open = false;
						query = '';
					}}
				>
					<span class="flex-1">
						<span class="block font-semibold">{e.name}</span>
						<span class="block text-xs text-muted">
							{e.unilateral ? 'einseitig · ' : ''}{e.defaultWeightKg ? `${e.defaultWeightKg} kg` : 'Körpergewicht'}
						</span>
					</span>
					{#if e.poseProfile}<ScanEye class="size-4 text-accent" aria-label="Formerkennung" />{/if}
				</button>
			{/each}
		</div>
	{:else}
		<p class="py-8 text-center text-muted">Keine Übung gefunden.</p>
	{/each}
</Sheet>

<script lang="ts">
	// Gewichtsauswahl: Schnellwahl der eigenen Kettlebells, Körpergewicht oder frei.
	import { app } from '#lib/state.svelte.ts';
	import type { Weight } from '#lib/types.ts';

	let { value = $bindable() }: { value: Weight } = $props();

	const presets = $derived(app.settings.weights);
	const isCustom = $derived(typeof value === 'number' && !presets.includes(value));
	let customOpen = $state(false);
</script>

<div>
	<span class="label">Gewicht</span>
	<div class="flex flex-wrap gap-2">
		{#each presets as w (w)}
			<button type="button" class="chip min-w-14 {value === w ? 'chip-active' : ''}" onclick={() => ((value = w), (customOpen = false))}>
				{w} kg
			</button>
		{/each}
		<button type="button" class="chip {value === 'bw' ? 'chip-active' : ''}" onclick={() => ((value = 'bw'), (customOpen = false))}>
			Körpergewicht
		</button>
		<button type="button" class="chip {isCustom || customOpen ? 'chip-active' : ''}" onclick={() => (customOpen = true)}>
			{isCustom ? `${value} kg` : 'Andere'}
		</button>
		{#if value !== null}
			<button type="button" class="chip text-muted" onclick={() => ((value = null), (customOpen = false))}>Ohne</button>
		{/if}
	</div>
	{#if customOpen}
		<div class="mt-2 flex items-center gap-2">
			<input
				class="input w-28"
				type="number"
				inputmode="decimal"
				min="0"
				max="200"
				step="0.5"
				value={typeof value === 'number' ? value : ''}
				oninput={(e) => {
					const v = parseFloat((e.currentTarget as HTMLInputElement).value);
					if (!Number.isNaN(v)) value = v;
				}}
			/>
			<span class="text-muted">kg</span>
		</div>
	{/if}
</div>

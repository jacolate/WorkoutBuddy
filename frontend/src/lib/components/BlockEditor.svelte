<script lang="ts">
	import { untrack } from 'svelte';
	import type { Block } from '#lib/types.ts';
	import { BLOCK_TYPE_LABELS } from '#lib/timer/compile.ts';
	import { newItem } from '#lib/workout.ts';
	import Stepper from './Stepper.svelte';
	import ItemEditor from './ItemEditor.svelte';
	import ExercisePicker from './ExercisePicker.svelte';
	import { ArrowUp, ArrowDown, Trash, Plus, Video, VideoOff } from '@lucide/svelte';

	let {
		block = $bindable(),
		index,
		onremove,
		onup,
		ondown,
		autoExpand = false
	}: {
		block: Block;
		index: number;
		onremove: () => void;
		onup?: () => void;
		ondown?: () => void;
		autoExpand?: boolean;
	} = $props();

	let picking = $state(false);
	// Nur der Anfangswert zählt: neu hinzugefügte Blöcke starten aufgeklappt.
	let expandedItem = $state<number | null>(untrack(() => autoExpand) ? 0 : null);

	const BADGE: Record<Block['type'], string> = {
		sets: 'bg-accent-soft text-accent',
		circuit: 'bg-accent-soft text-accent',
		interval: 'bg-warn-soft text-warn',
		emom: 'bg-warn-soft text-warn',
		amrap: 'bg-danger-soft text-danger',
		rest: 'bg-rest-soft text-rest'
	};

	function move<T>(arr: T[], i: number, d: number) {
		const j = i + d;
		if (j < 0 || j >= arr.length) return;
		[arr[i], arr[j]] = [arr[j], arr[i]];
		if (expandedItem === i) expandedItem = j;
		else if (expandedItem === j) expandedItem = i;
	}

	const HELP: Record<Block['type'], string> = {
		sets: 'Gleiche Übung mehrere Sätze mit Pause.',
		circuit: 'Mehrere Übungen nacheinander, mehrere Runden.',
		interval: 'Feste Arbeits-/Pausenzeiten, z. B. Tabata 8 × 20/10.',
		emom: 'Jede Minute startet eine Übung (rotierend), Rest der Minute ist Pause.',
		amrap: 'So viele Runden wie möglich in der vorgegebenen Zeit.',
		rest: 'Reine Pause.'
	};
</script>

<section class="card overflow-hidden">
	<header class="flex items-center gap-1 border-b border-line py-1.5 pr-1.5 pl-3">
		<span class="mr-1 rounded-md px-2 py-0.5 text-xs font-bold {BADGE[block.type]}">{index + 1} · {BLOCK_TYPE_LABELS[block.type]}</span>
		<input
			class="min-w-0 flex-1 bg-transparent px-1 py-2 font-semibold outline-none placeholder:font-normal placeholder:text-muted"
			placeholder="Titel (optional)"
			bind:value={block.title}
		/>
		{#if block.type !== 'rest'}
			<button
				type="button"
				class="icon-btn size-10 {block.record ? 'text-accent' : 'text-muted'}"
				onclick={() => (block.record = !block.record)}
				aria-label="Automatisch aufnehmen"
				title={block.record ? 'Wird automatisch aufgenommen (wenn Kamera an)' : 'Nicht automatisch aufnehmen'}
			>
				{#if block.record}<Video class="size-5" />{:else}<VideoOff class="size-5" />{/if}
			</button>
		{/if}
		{#if onup}<button type="button" class="icon-btn size-10" onclick={onup} aria-label="Block nach oben"><ArrowUp class="size-5" /></button>{/if}
		{#if ondown}<button type="button" class="icon-btn size-10" onclick={ondown} aria-label="Block nach unten"><ArrowDown class="size-5" /></button>{/if}
		<button type="button" class="icon-btn size-10 text-danger" onclick={onremove} aria-label="Block löschen"><Trash class="size-5" /></button>
	</header>

	<div class="space-y-4 p-3">
		<p class="text-xs text-muted">{HELP[block.type]}{block.record ? ' · 🎥 Arbeitsphasen werden aufgenommen.' : ''}</p>

		{#if block.type === 'sets'}
			<ItemEditor bind:item={block.item} expanded={autoExpand} />
			<div class="grid grid-cols-2 gap-3">
				<Stepper label="Sätze" bind:value={block.sets} min={1} max={50} />
				<Stepper label="Pause" bind:value={block.restSec} min={0} max={900} step={15} time />
			</div>
		{:else if block.type === 'rest'}
			<Stepper label="Dauer" bind:value={block.durationSec} min={5} max={1800} step={15} time />
		{:else}
			{#if block.type === 'circuit'}
				<div class="grid grid-cols-3 gap-2">
					<Stepper label="Runden" bind:value={block.rounds} min={1} max={50} />
					<Stepper label="Pause Übung" bind:value={block.restBetweenItemsSec} min={0} max={600} step={5} time />
					<Stepper label="Pause Runde" bind:value={block.restBetweenRoundsSec} min={0} max={900} step={15} time />
				</div>
			{:else if block.type === 'interval'}
				<div class="grid grid-cols-3 gap-2">
					<Stepper label="Runden" bind:value={block.rounds} min={1} max={100} />
					<Stepper label="Arbeit" bind:value={block.workSec} min={5} max={600} step={5} time />
					<Stepper label="Pause" bind:value={block.restSec} min={0} max={600} step={5} time />
				</div>
			{:else if block.type === 'emom'}
				<Stepper label="Minuten" bind:value={block.minutes} min={1} max={90} unit="min" />
			{:else if block.type === 'amrap'}
				<Stepper label="Zeitlimit" bind:value={block.durationSec} min={60} max={3600} step={60} time />
			{/if}

			<div class="space-y-2">
				{#each block.items as _, i (i)}
					<ItemEditor
						bind:item={block.items[i]}
						amount={block.type === 'interval' ? 'none' : block.type === 'circuit' ? 'repsOrTime' : 'reps'}
						expanded={expandedItem === i}
						onremove={() => {
							block.items.splice(i, 1);
							expandedItem = null;
						}}
						onup={i > 0 ? () => move(block.items, i, -1) : undefined}
						ondown={i < block.items.length - 1 ? () => move(block.items, i, 1) : undefined}
					/>
				{/each}
				<button type="button" class="btn-ghost w-full border border-dashed border-line text-muted" onclick={() => (picking = true)}>
					<Plus class="size-5" /> Übung hinzufügen
				</button>
			</div>
		{/if}
	</div>
</section>

{#if block.type !== 'sets' && block.type !== 'rest'}
	<ExercisePicker
		bind:open={picking}
		onpick={(e) => {
			block.items.push(newItem(e, block));
			expandedItem = block.items.length - 1;
		}}
	/>
{/if}

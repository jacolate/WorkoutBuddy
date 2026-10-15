<script lang="ts">
	// Bearbeitet eine Übung innerhalb eines Blocks (Übung, Reps/Zeit, Gewicht).
	import { app } from '#lib/state.svelte.ts';
	import type { BlockItem } from '#lib/types.ts';
	import { formatWeightShort } from '#lib/format.ts';
	import Stepper from './Stepper.svelte';
	import WeightPicker from './WeightPicker.svelte';
	import ExercisePicker from './ExercisePicker.svelte';
	import { ChevronDown, ChevronUp, X, ArrowUp, ArrowDown, ScanEye } from '@lucide/svelte';

	let {
		item = $bindable(),
		amount = 'repsOrTime',
		expanded = $bindable(false),
		onremove,
		onup,
		ondown
	}: {
		item: BlockItem;
		/** reps = nur Wiederholungen, repsOrTime = umschaltbar, none = keine Menge (Intervall) */
		amount?: 'reps' | 'repsOrTime' | 'none';
		expanded?: boolean;
		onremove?: () => void;
		onup?: () => void;
		ondown?: () => void;
	} = $props();

	let picking = $state(false);
	const exercise = $derived(app.exerciseMap.get(item.exerciseId));
	type Mode = 'reps' | 'time' | 'distance';
	const mode = $derived<Mode>(
		amount !== 'repsOrTime' || item.reps ? 'reps' : item.distanceM ? 'distance' : item.durationSec ? 'time' : 'reps'
	);
	const MODES: { id: Mode; label: string }[] = [
		{ id: 'reps', label: 'Wdh.' },
		{ id: 'time', label: 'Zeit' },
		{ id: 'distance', label: 'Strecke' }
	];

	function setMode(m: Mode) {
		item.reps = m === 'reps' ? item.reps || 10 : null;
		item.durationSec = m === 'time' ? item.durationSec || 30 : null;
		item.distanceM = m === 'distance' ? item.distanceM || 30 : null;
	}

	const summary = $derived.by(() => {
		const parts: string[] = [];
		if (amount !== 'none') {
			if (item.reps) parts.push(`${item.reps} Wdh.`);
			else if (item.distanceM) parts.push(`${item.distanceM} m`);
			else if (item.durationSec) parts.push(`${item.durationSec} s`);
		}
		if (item.weight !== null) parts.push(item.weight === 'bw' ? 'Körpergewicht' : `${formatWeightShort(item.weight)} kg`);
		if (exercise?.unilateral) parts.push('je Seite');
		return parts.join(' · ');
	});
</script>

<div class="rounded-xl border border-line bg-bg/50">
	<div class="flex items-center">
		<button type="button" class="flex min-h-14 flex-1 items-center gap-2 px-3 text-left" onclick={() => (expanded = !expanded)}>
			<span class="min-w-0 flex-1">
				<span class="flex items-center gap-1.5 font-semibold">
					<span class="truncate">{exercise?.name ?? 'Übung wählen'}</span>
					{#if exercise?.poseProfile}<ScanEye class="size-3.5 shrink-0 text-accent" />{/if}
				</span>
				<span class="block text-xs text-muted">{summary}</span>
			</span>
			{#if expanded}<ChevronUp class="size-5 text-muted" />{:else}<ChevronDown class="size-5 text-muted" />{/if}
		</button>
		{#if onup}<button type="button" class="icon-btn size-10" onclick={onup} aria-label="Nach oben"><ArrowUp class="size-4" /></button>{/if}
		{#if ondown}<button type="button" class="icon-btn size-10" onclick={ondown} aria-label="Nach unten"><ArrowDown class="size-4" /></button>{/if}
		{#if onremove}<button type="button" class="icon-btn size-10 text-muted" onclick={onremove} aria-label="Entfernen"><X class="size-4" /></button>{/if}
	</div>

	{#if expanded}
		<div class="space-y-4 border-t border-line p-3">
			<button type="button" class="btn-secondary w-full justify-between" onclick={() => (picking = true)}>
				<span>{exercise?.name ?? 'Übung wählen'}</span><span class="text-sm text-muted">ändern</span>
			</button>

			{#if amount === 'repsOrTime'}
				<div class="grid grid-cols-3 gap-1 rounded-xl bg-surface-2 p-1">
					{#each MODES as m (m.id)}
						<button type="button" class="h-10 rounded-lg text-sm font-semibold {mode === m.id ? 'bg-surface shadow-card' : 'text-muted'}" onclick={() => setMode(m.id)}>
							{m.label}
						</button>
					{/each}
				</div>
			{/if}

			{#if amount !== 'none' && mode === 'reps'}
				<Stepper label={exercise?.unilateral ? 'Wiederholungen (je Seite)' : 'Wiederholungen'} bind:value={() => item.reps ?? 10, (v) => (item.reps = v)} min={1} max={200} />
			{:else if mode === 'time'}
				<Stepper label={exercise?.unilateral ? 'Dauer (je Seite)' : 'Dauer'} bind:value={() => item.durationSec ?? 30, (v) => (item.durationSec = v)} min={5} max={3600} step={5} time />
			{:else if mode === 'distance'}
				<Stepper label={exercise?.unilateral ? 'Strecke (je Seite)' : 'Strecke'} bind:value={() => item.distanceM ?? 30, (v) => (item.distanceM = v)} min={5} max={1000} step={5} unit="m" />
			{/if}

			<WeightPicker bind:value={item.weight} />
		</div>
	{/if}
</div>

<ExercisePicker
	bind:open={picking}
	onpick={(e) => {
		item.exerciseId = e.id;
		item.weight = e.defaultWeightKg ?? 'bw';
	}}
/>

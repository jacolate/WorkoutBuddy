<script lang="ts">
	import { beforeNavigate, goto } from '$app/navigation';
	import { api } from '#lib/api.ts';
	import { app } from '#lib/state.svelte.ts';
	import type { Block, BlockType, Exercise, Workout, WorkoutStructure } from '#lib/types.ts';
	import { BLOCK_TYPE_LABELS, plannedVolume } from '#lib/timer/compile.ts';
	import { emptyStructure, newBlock, workoutSummary } from '#lib/workout.ts';
	import { formatDuration } from '#lib/format.ts';
	import BlockEditor from './BlockEditor.svelte';
	import ExercisePicker from './ExercisePicker.svelte';
	import Stepper from './Stepper.svelte';
	import Sheet from './Sheet.svelte';
	import { ChevronLeft, Play, Plus, Check, Repeat, ListChecks, Timer, Flame, Hourglass, Pause } from '@lucide/svelte';

	let { workout }: { workout: Workout | null } = $props();

	const initial = () => ({
		name: workout?.name ?? '',
		notes: workout?.notes ?? '',
		structure: workout ? (structuredClone(workout.structure) as WorkoutStructure) : emptyStructure(app.settings.prepareSec)
	});
	const start = initial();
	let name = $state(start.name);
	let notes = $state(start.notes);
	let structure = $state<WorkoutStructure>(start.structure);
	let savedSnapshot = $state(JSON.stringify(start));
	let saving = $state(false);
	let error = $state<string | null>(null);
	let addOpen = $state(false);
	let pendingType = $state<BlockType | null>(null);
	let picking = $state(false);
	let lastAdded = $state<string | null>(null);

	const dirty = $derived(JSON.stringify({ name, notes, structure }) !== savedSnapshot);
	const summary = $derived(workoutSummary(structure, app.exerciseMap, app.settings.secondsPerRep));
	const volume = $derived(plannedVolume(summary.steps));

	beforeNavigate((nav) => {
		if (!dirty || saving) return;
		// Tab schließen/neu laden: der Browser zeigt seinen eigenen Dialog.
		if (nav.type === 'leave') {
			nav.cancel();
			return;
		}
		if (!confirm('Ungespeicherte Änderungen verwerfen?')) nav.cancel();
	});

	const TYPES: { type: BlockType; icon: typeof Repeat; desc: string }[] = [
		{ type: 'sets', icon: Repeat, desc: 'Eine Übung, mehrere Sätze' },
		{ type: 'circuit', icon: ListChecks, desc: 'Mehrere Übungen im Wechsel' },
		{ type: 'interval', icon: Timer, desc: 'Arbeit/Pause nach Zeit (Tabata)' },
		{ type: 'emom', icon: Hourglass, desc: 'Every Minute On the Minute' },
		{ type: 'amrap', icon: Flame, desc: 'Max. Runden in fester Zeit' },
		{ type: 'rest', icon: Pause, desc: 'Längere Pause' }
	];

	function chooseType(type: BlockType) {
		addOpen = false;
		if (type === 'rest') {
			addBlock(newBlock('rest'));
			return;
		}
		pendingType = type;
		// Sheet erst schließen lassen, dann Übungsauswahl öffnen
		setTimeout(() => (picking = true), 50);
	}

	function addBlock(b: Block) {
		structure.blocks.push(b);
		lastAdded = b.id;
		setTimeout(() => document.getElementById(`block-${b.id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' }), 50);
	}

	function onPick(e: Exercise) {
		if (pendingType) addBlock(newBlock(pendingType, e));
		pendingType = null;
	}

	function moveBlock(i: number, d: number) {
		const j = i + d;
		const b = structure.blocks;
		[b[i], b[j]] = [b[j], b[i]];
	}

	async function save(thenPlay = false) {
		error = null;
		if (!name.trim()) {
			error = 'Bitte einen Namen eingeben.';
			document.getElementById('wname')?.focus();
			return;
		}
		saving = true;
		try {
			const payload = { name: name.trim(), notes, structure: $state.snapshot(structure) as WorkoutStructure };
			const saved = workout ? await api.updateWorkout({ id: workout.id, ...payload }) : await api.createWorkout(payload);
			savedSnapshot = JSON.stringify({ name: payload.name, notes, structure: payload.structure });
			name = payload.name;
			if (thenPlay) await goto(`/play/${saved.id}`);
			else if (!workout) await goto(`/workouts/${saved.id}`, { replaceState: true });
		} catch (err) {
			error = (err as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head><title>{name || 'Neues Workout'} · Workout</title></svelte:head>

<div class="sticky top-0 z-20 -mx-4 mb-4 flex items-center gap-2 border-b border-line bg-bg/90 px-2 py-2 backdrop-blur-lg md:-mx-8 md:px-6">
	<a href="/" class="icon-btn" aria-label="Zurück"><ChevronLeft class="size-6" /></a>
	<div class="min-w-0 flex-1 text-sm text-muted">
		<span class="tabular">~{formatDuration(summary.durationSec)}</span> · {summary.blocks} Blöcke{volume.reps ? ` · ${volume.reps} Wdh.` : ''}
	</div>
	{#if workout && !dirty}
		<a href="/play/{workout.id}" class="btn-primary h-11 min-h-11 px-4 {summary.steps.length === 0 ? 'pointer-events-none opacity-50' : ''}">
			<Play class="size-4 fill-current" /> Start
		</a>
	{:else}
		<button class="btn-primary h-11 min-h-11 px-4" onclick={() => save()} disabled={saving}>
			<Check class="size-5" />
			{saving ? 'Speichert …' : 'Speichern'}
		</button>
	{/if}
</div>

<div class="space-y-4">
	<input
		id="wname"
		class="w-full bg-transparent text-2xl font-bold tracking-tight outline-none placeholder:text-muted md:text-3xl"
		placeholder="Name des Workouts"
		bind:value={name}
	/>
	<textarea
		class="input min-h-20 py-2 text-sm"
		rows="2"
		placeholder="Notizen (optional)"
		bind:value={notes}
	></textarea>

	{#if error}<p class="rounded-xl bg-danger-soft px-3 py-2 text-sm font-medium text-danger">{error}</p>{/if}

	<div class="grid grid-cols-2 gap-3">
		<Stepper label="Vorbereitung" bind:value={structure.prepareSec} min={0} max={120} step={5} time />
		<Stepper label="Pause zw. Blöcken" bind:value={structure.blockRestSec} min={0} max={600} step={15} time />
	</div>

	{#each structure.blocks as block, i (block.id)}
		<div id="block-{block.id}" class="scroll-mt-20">
			<BlockEditor
				bind:block={structure.blocks[i]}
				index={i}
				autoExpand={lastAdded === block.id}
				onremove={() => {
					if (confirm('Block löschen?')) structure.blocks.splice(i, 1);
				}}
				onup={i > 0 ? () => moveBlock(i, -1) : undefined}
				ondown={i < structure.blocks.length - 1 ? () => moveBlock(i, 1) : undefined}
			/>
		</div>
	{/each}

	<button class="btn-secondary w-full border-dashed" onclick={() => (addOpen = true)}>
		<Plus class="size-5" /> Block hinzufügen
	</button>

	{#if dirty || !workout}
		<div class="grid gap-2 pt-2 sm:grid-cols-2">
			<button class="btn-primary" onclick={() => save()} disabled={saving}><Check class="size-5" /> Speichern</button>
			<button class="btn-secondary" onclick={() => save(true)} disabled={saving || summary.steps.length === 0}>
				<Play class="size-5" /> Speichern & starten
			</button>
		</div>
	{/if}
</div>

<Sheet bind:open={addOpen} title="Block hinzufügen">
	<div class="grid gap-2">
		{#each TYPES as t (t.type)}
			<button class="flex min-h-16 items-center gap-3 rounded-xl border border-line px-3 text-left hover:bg-surface-2 active:bg-surface-2" onclick={() => chooseType(t.type)}>
				<span class="grid size-10 place-items-center rounded-lg bg-accent-soft text-accent"><t.icon class="size-5" /></span>
				<span>
					<span class="block font-semibold">{BLOCK_TYPE_LABELS[t.type]}</span>
					<span class="block text-sm text-muted">{t.desc}</span>
				</span>
			</button>
		{/each}
	</div>
</Sheet>

<ExercisePicker bind:open={picking} onpick={onPick} />

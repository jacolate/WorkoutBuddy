<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api } from '#lib/api.ts';
	import { app } from '#lib/state.svelte.ts';
	import type { Session, Workout } from '#lib/types.ts';
	import { describeBlock, workoutSummary } from '#lib/workout.ts';
	import { formatDuration, relativeDay, summarizeLog } from '#lib/format.ts';
	import { Play, Plus, Pencil, Copy, Trash, Timer, ChevronRight } from '@lucide/svelte';

	let workouts = $state<Workout[]>([]);
	let sessions = $state<Session[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		try {
			[workouts, sessions] = await Promise.all([api.workouts(), api.sessions()]);
		} catch (err) {
			error = (err as Error).message;
		} finally {
			loading = false;
		}
	}
	onMount(load);

	const last = $derived(sessions[0]);
	const thisWeek = $derived.by(() => {
		const now = new Date();
		const monday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - ((now.getDay() + 6) % 7));
		return sessions.filter((s) => new Date(s.startedAt) >= monday && s.finishedAt).length;
	});

	async function duplicate(w: Workout) {
		const copy = await api.duplicateWorkout(w.id);
		await goto(`/workouts/${copy.id}`);
	}

	async function remove(w: Workout) {
		if (!confirm(`„${w.name}“ löschen? Der Verlauf bleibt erhalten.`)) return;
		await api.deleteWorkout(w.id);
		workouts = workouts.filter((x) => x.id !== w.id);
	}

	function greeting() {
		const h = new Date().getHours();
		if (h < 11) return 'Guten Morgen';
		if (h < 18) return 'Hallo';
		return 'Guten Abend';
	}
</script>

<svelte:head><title>Training · Workout</title></svelte:head>

<header class="mb-6 flex items-end justify-between gap-4">
	<div>
		<p class="text-sm font-medium text-muted">{greeting()}{app.user ? `, ${app.user}` : ''}</p>
		<h1 class="text-3xl font-bold tracking-tight">Training</h1>
	</div>
	<a href="/workouts/new" class="btn-primary shrink-0 px-3 sm:px-4">
		<Plus class="size-5" /><span class="hidden sm:inline">Neues Workout</span><span class="sm:hidden">Neu</span>
	</a>
</header>

{#if loading}
	<div class="space-y-3">
		{#each [1, 2, 3] as i (i)}<div class="card h-28 animate-pulse"></div>{/each}
	</div>
{:else if error}
	<p class="card p-4 text-danger">{error}</p>
{:else}
	{#if last}
		{@const sum = summarizeLog(last.log)}
		<a href="/history/{last.id}" class="card mb-6 flex items-center gap-4 p-4 transition hover:bg-surface-2">
			<div class="grid size-12 shrink-0 place-items-center rounded-xl bg-ok-soft text-ok">
				<Timer class="size-6" />
			</div>
			<div class="min-w-0 flex-1">
				<p class="text-xs font-semibold tracking-wide text-muted uppercase">Letztes Training · {relativeDay(last.startedAt)}</p>
				<p class="truncate font-semibold">{last.workoutName || 'Workout'}</p>
				<p class="text-sm text-muted">
					{sum.reps} Wdh. · {Math.round(sum.volumeKg)} kg bewegt · {thisWeek}× diese Woche
				</p>
			</div>
			<ChevronRight class="size-5 text-muted" />
		</a>
	{/if}

	<div class="space-y-3">
		{#each workouts as w (w.id)}
			{@const sum = workoutSummary(w.structure, app.exerciseMap, app.settings.secondsPerRep)}
			<article class="card overflow-hidden">
				<a href="/workouts/{w.id}" class="block p-4 pb-2">
					<div class="flex items-start justify-between gap-3">
						<h2 class="text-lg leading-tight font-bold">{w.name}</h2>
						<span class="tabular shrink-0 rounded-full bg-surface-2 px-2.5 py-1 text-xs font-semibold text-muted">
							~{formatDuration(sum.durationSec)}
						</span>
					</div>
					<ul class="mt-2 space-y-0.5 text-sm text-muted">
						{#each w.structure.blocks.slice(0, 4) as b (b.id)}
							<li class="truncate">
								<span class="font-medium text-fg">{b.title || ''}</span>{b.title ? ' · ' : ''}{describeBlock(b, app.exerciseMap)}
							</li>
						{/each}
						{#if w.structure.blocks.length > 4}<li>+ {w.structure.blocks.length - 4} weitere</li>{/if}
						{#if w.structure.blocks.length === 0}<li>Noch leer – tippen zum Bearbeiten</li>{/if}
					</ul>
				</a>
				<div class="flex items-center gap-1 px-2 pb-2">
					<a
						href="/play/{w.id}"
						class="btn-primary h-11 min-h-11 flex-1 {sum.steps.length === 0 ? 'pointer-events-none opacity-50' : ''}"
						aria-disabled={sum.steps.length === 0}
					>
						<Play class="size-5 fill-current" /> Starten
					</a>
					<a href="/workouts/{w.id}" class="icon-btn" aria-label="Bearbeiten"><Pencil class="size-5" /></a>
					<button class="icon-btn" onclick={() => duplicate(w)} aria-label="Duplizieren"><Copy class="size-5" /></button>
					<button class="icon-btn text-danger" onclick={() => remove(w)} aria-label="Löschen"><Trash class="size-5" /></button>
				</div>
				{#if w.lastPlayedAt}
					<p class="border-t border-line px-4 py-2 text-xs text-muted">Zuletzt: {relativeDay(w.lastPlayedAt)}</p>
				{/if}
			</article>
		{:else}
			<div class="card p-8 text-center">
				<p class="mb-4 text-muted">Noch keine Workouts.</p>
				<a href="/workouts/new" class="btn-primary"><Plus class="size-5" /> Erstes Workout anlegen</a>
			</div>
		{/each}
	</div>
{/if}

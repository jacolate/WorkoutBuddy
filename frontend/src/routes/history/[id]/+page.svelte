<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '#lib/api.ts';
	import type { LogEntry, Recording, Session } from '#lib/types.ts';
	import { blockLabel } from '#lib/timer/compile.ts';
	import { formatBytes, formatClock, formatDateTime, formatSide, formatWeight, summarizeLog } from '#lib/format.ts';
	import { ChevronLeft, Trash, Play, Video, RotateCcw } from '@lucide/svelte';

	const id = Number(page.params.id);
	let session = $state<Session | null>(null);
	let recordings = $state<Recording[]>([]);
	let error = $state<string | null>(null);
	let notes = $state('');
	let notesDirty = $state(false);

	onMount(async () => {
		try {
			[session, recordings] = await Promise.all([api.session(id), api.recordings(id)]);
			notes = session.notes;
		} catch (err) {
			error = (err as Error).message;
		}
	});

	const sum = $derived(session ? summarizeLog(session.log) : null);
	const dur = $derived(session?.finishedAt ? (new Date(session.finishedAt).getTime() - new Date(session.startedAt).getTime()) / 1000 : null);

	// Log nach Block gruppieren
	const blocks = $derived.by(() => {
		if (!session) return [];
		const m = new Map<number, LogEntry[]>();
		for (const e of session.log) {
			if (!m.has(e.blockIndex)) m.set(e.blockIndex, []);
			m.get(e.blockIndex)!.push(e);
		}
		return [...m.entries()].map(([bi, entries]) => ({
			label: session!.snapshot.blocks[bi] ? blockLabel(session!.snapshot.blocks[bi]) : `Block ${bi + 1}`,
			entries
		}));
	});

	async function saveNotes() {
		if (!session) return;
		session = await api.patchSession(session.id, { notes });
		notesDirty = false;
	}

	async function remove() {
		if (!session || !confirm('Training inkl. Aufnahmen löschen?')) return;
		await api.deleteSession(session.id);
		await goto('/history');
	}
</script>

<svelte:head><title>{session?.workoutName ?? 'Training'} · Verlauf</title></svelte:head>

<div class="mb-4 flex items-center gap-2">
	<a href="/history" class="icon-btn -ml-3" aria-label="Zurück"><ChevronLeft class="size-6" /></a>
	<div class="flex-1"></div>
	{#if session?.workoutId}
		<a href="/play/{session.workoutId}" class="btn-secondary h-11 min-h-11"><RotateCcw class="size-4" /> Nochmal</a>
	{/if}
	<button class="icon-btn text-danger" onclick={remove} aria-label="Löschen"><Trash class="size-5" /></button>
</div>

{#if error}
	<p class="card p-4 text-danger">{error}</p>
{:else if !session || !sum}
	<div class="card h-40 animate-pulse"></div>
{:else}
	<h1 class="text-3xl font-bold tracking-tight">{session.workoutName || 'Workout'}</h1>
	<p class="mb-5 text-muted">{formatDateTime(session.startedAt)}</p>

	<div class="mb-6 grid grid-cols-3 gap-2 text-center">
		<div class="card p-3">
			<p class="tabular text-xl font-black">{dur ? formatClock(dur) : '–'}</p>
			<p class="text-xs text-muted">Dauer</p>
		</div>
		<div class="card p-3">
			<p class="tabular text-xl font-black">{sum.reps}</p>
			<p class="text-xs text-muted">Wiederholungen</p>
		</div>
		<div class="card p-3">
			<p class="tabular text-xl font-black">{Math.round(sum.volumeKg).toLocaleString('de-DE')}</p>
			<p class="text-xs text-muted">kg bewegt</p>
		</div>
	</div>

	{#if recordings.length > 0}
		<h2 class="mb-2 text-lg font-bold">Aufnahmen</h2>
		<div class="mb-6 grid gap-2 sm:grid-cols-2">
			{#each recordings as r (r.id)}
				<a href="/recordings/{r.id}" class="card flex items-center gap-3 p-3 transition hover:bg-surface-2">
					<span class="grid size-12 shrink-0 place-items-center rounded-xl bg-accent-soft text-accent"><Play class="size-5 fill-current" /></span>
					<span class="min-w-0 flex-1">
						<span class="block truncate font-semibold">{r.label || r.exerciseName || 'Aufnahme'}</span>
						<span class="block text-xs text-muted">{formatClock(r.durationSec)} · {formatBytes(r.sizeBytes)}{r.note ? ' · 📝' : ''}</span>
					</span>
				</a>
			{/each}
		</div>
	{/if}

	<h2 class="mb-2 text-lg font-bold">Protokoll</h2>
	{#if blocks.length === 0}
		<p class="card mb-6 p-4 text-muted">Keine Einträge.</p>
	{/if}
	<div class="mb-6 space-y-3">
		{#each blocks as b, bi (bi)}
			<section class="card overflow-hidden">
				<h3 class="border-b border-line px-4 py-2 text-sm font-bold text-muted">{b.label}</h3>
				<ul class="divide-y divide-line">
					{#each b.entries as e, i (i)}
						<li class="flex items-center gap-3 px-4 py-2 text-sm {e.skipped ? 'opacity-50' : ''}">
							<span class="flex-1">
								<span class="font-medium">{e.exerciseName}</span>
								{#if e.side}<span class="text-muted"> · {formatSide(e.side)}</span>{/if}
								{#if e.rounds !== undefined}<span class="text-muted"> · {e.rounds} Runden</span>{/if}
								{#if e.skipped}<span class="text-muted"> · übersprungen</span>{/if}
							</span>
							<span class="tabular text-muted">{formatWeight(e.weight)}</span>
							<span class="tabular w-16 text-right font-semibold">{e.reps !== null ? `${e.reps}×` : e.distanceM ? `${e.distanceM} m` : formatClock(e.durationSec)}</span>
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	</div>

	<label class="label" for="snotes">Notizen</label>
	<textarea id="snotes" class="input min-h-24 py-2" bind:value={notes} oninput={() => (notesDirty = true)}></textarea>
	{#if notesDirty}
		<button class="btn-primary mt-2 w-full" onclick={saveNotes}>Notiz speichern</button>
	{/if}
{/if}

{#if recordings.length === 0 && session}
	<p class="mt-6 flex items-center gap-2 text-sm text-muted">
		<Video class="size-4" /> Tipp: Im Player mit Kamera kannst du Sätze aufnehmen und hier nachträglich ansehen.
	</p>
{/if}

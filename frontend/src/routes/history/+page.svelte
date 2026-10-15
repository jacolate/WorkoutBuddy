<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api.ts';
	import type { Session } from '#lib/types.ts';
	import { formatClock, formatDate, relativeDay, summarizeLog } from '#lib/format.ts';
	import { ChevronRight, Video, Flame } from '@lucide/svelte';

	let sessions = $state<Session[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			sessions = await api.sessions();
		} catch (err) {
			error = (err as Error).message;
		} finally {
			loading = false;
		}
	});

	function duration(s: Session) {
		if (!s.finishedAt) return null;
		return (new Date(s.finishedAt).getTime() - new Date(s.startedAt).getTime()) / 1000;
	}

	// Nach Monaten gruppieren
	const groups = $derived.by(() => {
		const fmt = new Intl.DateTimeFormat('de-DE', { month: 'long', year: 'numeric' });
		const m = new Map<string, Session[]>();
		for (const s of sessions) {
			const k = fmt.format(new Date(s.startedAt));
			if (!m.has(k)) m.set(k, []);
			m.get(k)!.push(s);
		}
		return [...m.entries()];
	});

	// Trainingstage der letzten 12 Wochen
	const streak = $derived.by(() => {
		const days = new Set(sessions.map((s) => new Date(s.startedAt).toDateString()));
		const weeks: { date: Date; active: boolean }[][] = [];
		const today = new Date();
		const start = new Date(today.getFullYear(), today.getMonth(), today.getDate() - ((today.getDay() + 6) % 7) - 7 * 11);
		for (let w = 0; w < 12; w++) {
			const week = [];
			for (let d = 0; d < 7; d++) {
				const date = new Date(start.getFullYear(), start.getMonth(), start.getDate() + w * 7 + d);
				week.push({ date, active: days.has(date.toDateString()) });
			}
			weeks.push(week);
		}
		return weeks;
	});
	const last30 = $derived(sessions.filter((s) => Date.now() - new Date(s.startedAt).getTime() < 30 * 86_400_000).length);
</script>

<svelte:head><title>Verlauf · Workout</title></svelte:head>

<h1 class="mb-6 text-3xl font-bold tracking-tight">Verlauf</h1>

{#if loading}
	<div class="space-y-3">{#each [1, 2, 3] as i (i)}<div class="card h-20 animate-pulse"></div>{/each}</div>
{:else if error}
	<p class="card p-4 text-danger">{error}</p>
{:else if sessions.length === 0}
	<div class="card p-8 text-center text-muted">Noch keine Trainings. Starte dein erstes Workout!</div>
{:else}
	<section class="card mb-6 p-4">
		<div class="mb-3 flex items-center justify-between">
			<p class="font-semibold">Letzte 12 Wochen</p>
			<p class="flex items-center gap-1 text-sm text-muted"><Flame class="size-4 text-accent" /> {last30} Trainings in 30 Tagen</p>
		</div>
		<div class="flex justify-between gap-1">
			{#each streak as week, wi (wi)}
				<div class="flex flex-1 flex-col gap-1">
					{#each week as day (day.date.toISOString())}
						<div
							class="aspect-square w-full rounded-[4px] {day.active ? 'bg-accent' : 'bg-surface-2'} {day.date > new Date() ? 'opacity-30' : ''}"
							title={formatDate(day.date.toISOString())}
						></div>
					{/each}
				</div>
			{/each}
		</div>
	</section>

	{#each groups as [month, list] (month)}
		<h2 class="mt-6 mb-2 px-1 text-xs font-bold tracking-wider text-muted uppercase">{month}</h2>
		<div class="card divide-y divide-line overflow-hidden">
			{#each list as s (s.id)}
				{@const sum = summarizeLog(s.log)}
				{@const dur = duration(s)}
				<a href="/history/{s.id}" class="flex items-center gap-3 px-4 py-3 transition hover:bg-surface-2">
					<div class="min-w-0 flex-1">
						<p class="truncate font-semibold">{s.workoutName || 'Workout'}</p>
						<p class="text-sm text-muted">
							{relativeDay(s.startedAt)}{dur ? ` · ${formatClock(dur)}` : ' · nicht beendet'} · {sum.reps} Wdh. · {Math.round(sum.volumeKg)} kg
						</p>
					</div>
					{#if s.recordingCount > 0}
						<span class="flex items-center gap-1 rounded-full bg-surface-2 px-2 py-0.5 text-xs font-semibold text-muted">
							<Video class="size-3.5" />{s.recordingCount}
						</span>
					{/if}
					<ChevronRight class="size-5 text-muted" />
				</a>
			{/each}
		</div>
	{/each}
{/if}

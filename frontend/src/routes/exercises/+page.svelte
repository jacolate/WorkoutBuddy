<script lang="ts">
	import { api } from '#lib/api.ts';
	import { app } from '#lib/state.svelte.ts';
	import type { Exercise } from '#lib/types.ts';
	import { CATEGORIES, categoryLabel } from '#lib/categories.ts';
	import { PROFILE_OPTIONS } from '#lib/pose/profiles.ts';
	import Sheet from '#lib/components/Sheet.svelte';
	import WeightPicker from '#lib/components/WeightPicker.svelte';
	import { Plus, Search, ScanEye, ChevronRight } from '@lucide/svelte';

	let query = $state('');
	let editing = $state<Exercise | null>(null);
	let open = $state(false);
	let saving = $state(false);
	let error = $state<string | null>(null);

	const filtered = $derived(app.exercises.filter((e) => !query || e.name.toLowerCase().includes(query.toLowerCase())));
	const groups = $derived.by(() => {
		const m = new Map<string, Exercise[]>();
		for (const e of filtered) {
			const k = categoryLabel(e.category);
			if (!m.has(k)) m.set(k, []);
			m.get(k)!.push(e);
		}
		return [...m.entries()].sort((a, b) => a[0].localeCompare(b[0], 'de'));
	});

	function edit(e: Exercise | null) {
		error = null;
		editing = e
			? { ...e }
			: { id: 0, name: '', description: '', category: 'swing', unilateral: false, defaultWeightKg: 16, poseProfile: null, cues: '', cameraHint: '' };
		open = true;
	}

	async function save() {
		if (!editing) return;
		saving = true;
		error = null;
		try {
			if (editing.id) await api.updateExercise(editing);
			else await api.createExercise(editing);
			await app.reloadExercises();
			open = false;
		} catch (err) {
			error = (err as Error).message;
		} finally {
			saving = false;
		}
	}

	async function remove() {
		if (!editing?.id || !confirm(`„${editing.name}“ löschen? Workouts, die sie verwenden, zeigen dann „Unbekannte Übung“.`)) return;
		await api.deleteExercise(editing.id);
		await app.reloadExercises();
		open = false;
	}
</script>

<svelte:head><title>Übungen · Workout</title></svelte:head>

<header class="mb-4 flex items-end justify-between gap-4">
	<h1 class="text-3xl font-bold tracking-tight">Übungen</h1>
	<button class="btn-primary px-3 sm:px-4" onclick={() => edit(null)}><Plus class="size-5" /> Neu</button>
</header>

<div class="relative mb-4">
	<Search class="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-muted" />
	<input class="input pl-10" type="search" placeholder="Suchen …" bind:value={query} />
</div>

<p class="mb-4 flex items-center gap-1.5 text-sm text-muted"><ScanEye class="size-4 text-accent" /> = Formerkennung & automatisches Zählen verfügbar</p>

{#each groups as [cat, list] (cat)}
	<h2 class="mt-5 mb-2 px-1 text-xs font-bold tracking-wider text-muted uppercase">{cat}</h2>
	<div class="card divide-y divide-line overflow-hidden">
		{#each list as e (e.id)}
			<button class="flex min-h-14 w-full items-center gap-3 px-4 py-2 text-left transition hover:bg-surface-2" onclick={() => edit(e)}>
				<span class="min-w-0 flex-1">
					<span class="flex items-center gap-1.5 font-semibold">{e.name}{#if e.poseProfile}<ScanEye class="size-4 text-accent" />{/if}</span>
					<span class="block truncate text-sm text-muted">{e.description}</span>
				</span>
				<span class="text-sm text-muted">{e.defaultWeightKg ? `${e.defaultWeightKg} kg` : ''}</span>
				<ChevronRight class="size-5 text-muted" />
			</button>
		{/each}
	</div>
{/each}

<Sheet bind:open title={editing?.id ? 'Übung bearbeiten' : 'Neue Übung'}>
	{#if editing}
		<div class="space-y-4">
			<div>
				<label class="label" for="ename">Name</label>
				<input id="ename" class="input" bind:value={editing.name} />
			</div>
			<div>
				<label class="label" for="ecat">Kategorie</label>
				<select id="ecat" class="input" bind:value={editing.category}>
					{#each CATEGORIES as c (c.id)}<option value={c.id}>{c.label}</option>{/each}
				</select>
			</div>
			<label class="flex min-h-12 items-center gap-3">
				<input type="checkbox" class="size-5 accent-[var(--accent)]" bind:checked={editing.unilateral} />
				<span>
					<span class="block font-medium">Einseitig</span>
					<span class="block text-xs text-muted">Wird im Training automatisch für links und rechts ausgeführt</span>
				</span>
			</label>
			<WeightPicker bind:value={() => editing!.defaultWeightKg ?? 'bw', (v) => (editing!.defaultWeightKg = typeof v === 'number' ? v : null)} />
			<div>
				<label class="label" for="epose">Formerkennung</label>
				<select id="epose" class="input" bind:value={editing.poseProfile}>
					<option value={null}>Keine (nur Skelett anzeigen)</option>
					{#each PROFILE_OPTIONS as o (o.id)}<option value={o.id}>{o.label}</option>{/each}
				</select>
			</div>
			<div>
				<label class="label" for="ecam">Kamera-Position</label>
				<select id="ecam" class="input" bind:value={editing.cameraHint}>
					<option value="">egal</option>
					<option value="seitlich">seitlich</option>
					<option value="frontal">frontal</option>
				</select>
			</div>
			<div>
				<label class="label" for="edesc">Beschreibung</label>
				<textarea id="edesc" class="input min-h-16 py-2" bind:value={editing.description}></textarea>
			</div>
			<div>
				<label class="label" for="ecues">Ausführungshinweise (eine pro Zeile)</label>
				<textarea id="ecues" class="input min-h-28 py-2" bind:value={editing.cues}></textarea>
			</div>
			{#if error}<p class="rounded-xl bg-danger-soft px-3 py-2 text-sm text-danger">{error}</p>{/if}
		</div>
	{/if}
	{#snippet footer()}
		<div class="flex gap-2">
			{#if editing?.id}<button class="btn-danger" onclick={remove}>Löschen</button>{/if}
			<button class="btn-primary flex-1" onclick={save} disabled={saving || !editing?.name.trim()}>{saving ? 'Speichert …' : 'Speichern'}</button>
		</div>
	{/snippet}
</Sheet>

<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '#lib/api.ts';
	import { app, theme, type ThemePref } from '#lib/state.svelte.ts';
	import { uploads } from '#lib/camera/uploads.svelte.ts';
	import { beeps, speak, unlockAudio } from '#lib/audio/audio.ts';
	import { wakeLockSupported } from '#lib/wakelock.ts';
	import { canRecord } from '#lib/camera/camera.ts';
	import Stepper from '#lib/components/Stepper.svelte';
	import { Sun, Moon, Monitor, LogOut, X, Plus, CircleCheck, CircleAlert, RotateCcw, Download } from '@lucide/svelte';

	let newWeight = $state(20);
	let saveError = $state<string | null>(null);

	const themes: { id: ThemePref; label: string; icon: typeof Sun }[] = [
		{ id: 'system', label: 'System', icon: Monitor },
		{ id: 'light', label: 'Hell', icon: Sun },
		{ id: 'dark', label: 'Dunkel', icon: Moon }
	];

	async function save(patch: Parameters<typeof app.saveSettings>[0]) {
		saveError = null;
		try {
			await app.saveSettings(patch);
		} catch (err) {
			saveError = (err as Error).message;
		}
	}

	function addWeight() {
		if (newWeight <= 0 || app.settings.weights.includes(newWeight)) return;
		void save({ weights: [...app.settings.weights, newWeight].sort((a, b) => a - b) });
	}

	async function logout() {
		await api.logout().catch(() => {});
		app.reset();
		await goto('/login', { replaceState: true });
	}

	const checks = [
		{ label: 'Sichere Verbindung (HTTPS)', ok: window.isSecureContext, hint: 'Nötig für Kamera und Bildschirm-an. App über die HTTPS-Adresse des Reverse-Proxys öffnen.' },
		{ label: 'Kamera-API', ok: !!navigator.mediaDevices?.getUserMedia, hint: 'Nur über HTTPS verfügbar.' },
		{ label: 'Videoaufnahme', ok: canRecord(), hint: 'Browser unterstützt MediaRecorder nicht.' },
		{ label: 'Bildschirm bleibt an', ok: wakeLockSupported(), hint: 'Wake-Lock nicht verfügbar – ggf. Auto-Sperre manuell verlängern.' },
		{ label: 'Sprachausgabe', ok: 'speechSynthesis' in window, hint: 'Keine Sprachausgabe verfügbar.' }
	];
</script>

<svelte:head><title>Einstellungen · Workout</title></svelte:head>

<h1 class="mb-6 text-3xl font-bold tracking-tight">Einstellungen</h1>

{#if saveError}<p class="mb-4 rounded-xl bg-danger-soft px-3 py-2 text-sm text-danger">{saveError}</p>{/if}

<div class="space-y-6">
	<section>
		<h2 class="mb-2 px-1 text-xs font-bold tracking-wider text-muted uppercase">Darstellung</h2>
		<div class="card grid grid-cols-3 gap-1 p-1">
			{#each themes as t (t.id)}
				<button
					class="flex min-h-12 items-center justify-center gap-2 rounded-xl font-semibold transition {theme.pref === t.id ? 'bg-accent text-accent-fg' : 'text-muted hover:bg-surface-2'}"
					onclick={() => theme.set(t.id)}
				>
					<t.icon class="size-4" />{t.label}
				</button>
			{/each}
		</div>
	</section>

	<section>
		<h2 class="mb-2 px-1 text-xs font-bold tracking-wider text-muted uppercase">Meine Kettlebells</h2>
		<div class="card p-4">
			<div class="mb-3 flex flex-wrap gap-2">
				{#each app.settings.weights as w (w)}
					<span class="chip gap-1 pr-1.5">
						{w} kg
						<button
							class="grid size-7 place-items-center rounded-full text-muted hover:bg-surface-2"
							onclick={() => save({ weights: app.settings.weights.filter((x) => x !== w) })}
							aria-label="{w} kg entfernen"
							disabled={app.settings.weights.length <= 1}><X class="size-4" /></button
						>
					</span>
				{/each}
			</div>
			<div class="flex items-end gap-2">
				<div class="flex-1"><Stepper bind:value={newWeight} min={1} max={100} unit="kg" /></div>
				<button class="btn-secondary" onclick={addWeight}><Plus class="size-5" /> Hinzufügen</button>
			</div>
		</div>
	</section>

	<section>
		<h2 class="mb-2 px-1 text-xs font-bold tracking-wider text-muted uppercase">Training</h2>
		<div class="card divide-y divide-line">
			<label class="flex min-h-14 items-center gap-3 px-4">
				<span class="flex-1 font-medium">Töne (Countdown, Start, Pause)</span>
				<input type="checkbox" class="size-5 accent-[var(--accent)]" checked={app.settings.sound} onchange={(e) => save({ sound: e.currentTarget.checked })} />
			</label>
			<label class="flex min-h-14 items-center gap-3 px-4">
				<span class="flex-1 font-medium">Sprachansagen</span>
				<input type="checkbox" class="size-5 accent-[var(--accent)]" checked={app.settings.voice} onchange={(e) => save({ voice: e.currentTarget.checked })} />
			</label>
			<label class="flex min-h-14 items-center gap-3 px-4 py-2">
				<span class="flex-1">
					<span class="block font-medium">Automatisch weiter bei erkannten Reps</span>
					<span class="block text-xs text-muted">Wenn die Formerkennung das Wiederholungsziel zählt, geht es automatisch weiter.</span>
				</span>
				<input type="checkbox" class="size-5 accent-[var(--accent)]" checked={app.settings.autoAdvanceReps} onchange={(e) => save({ autoAdvanceReps: e.currentTarget.checked })} />
			</label>
			<div class="grid grid-cols-2 gap-3 p-4">
				<Stepper label="Vorbereitung (neue Workouts)" bind:value={() => app.settings.prepareSec, (v) => save({ prepareSec: v })} min={0} max={60} step={5} time />
				<Stepper label="Sekunden pro Wdh. (Schätzung)" bind:value={() => app.settings.secondsPerRep, (v) => save({ secondsPerRep: v })} min={1} max={10} unit="s" />
			</div>
			<div class="flex gap-2 p-4">
				<button
					class="btn-secondary flex-1"
					onclick={() => {
						unlockAudio();
						beeps.tick();
						setTimeout(() => beeps.go(), 400);
					}}>Ton testen</button
				>
				<button
					class="btn-secondary flex-1"
					onclick={() => {
						unlockAudio();
						speak('Goblet Squat, 16 Kilo, 10 Wiederholungen');
					}}>Stimme testen</button
				>
			</div>
		</div>
	</section>

	<section id="uploads">
		<h2 class="mb-2 px-1 text-xs font-bold tracking-wider text-muted uppercase">Gerät</h2>
		<div class="card divide-y divide-line">
			{#each checks as c (c.label)}
				<div class="flex items-start gap-3 px-4 py-3">
					{#if c.ok}<CircleCheck class="mt-0.5 size-5 shrink-0 text-ok" />{:else}<CircleAlert class="mt-0.5 size-5 shrink-0 text-warn" />{/if}
					<div>
						<p class="font-medium">{c.label}</p>
						{#if !c.ok}<p class="text-sm text-muted">{c.hint}</p>{/if}
					</div>
				</div>
			{/each}
		</div>
		{#if uploads.failed.length > 0}
			<div class="card mt-3 divide-y divide-line">
				{#each uploads.failed as j (j.id)}
					<div class="flex items-center gap-2 px-4 py-3">
						<div class="min-w-0 flex-1">
							<p class="truncate font-medium">{j.label}</p>
							<p class="text-xs text-danger">{j.error}</p>
						</div>
						<button class="icon-btn" onclick={() => uploads.retry(j.id)} aria-label="Erneut versuchen"><RotateCcw class="size-5" /></button>
						<button class="icon-btn" onclick={() => uploads.download(j.id)} aria-label="Lokal speichern"><Download class="size-5" /></button>
						<button class="icon-btn text-danger" onclick={() => uploads.discard(j.id)} aria-label="Verwerfen"><X class="size-5" /></button>
					</div>
				{/each}
			</div>
		{/if}
	</section>

	<section>
		<h2 class="mb-2 px-1 text-xs font-bold tracking-wider text-muted uppercase">Konto</h2>
		<div class="card flex items-center gap-3 p-4">
			<p class="flex-1">Angemeldet als <strong>{app.user}</strong></p>
			<button class="btn-secondary" onclick={logout}><LogOut class="size-5" /> Abmelden</button>
		</div>
	</section>
</div>

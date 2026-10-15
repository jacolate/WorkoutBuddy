<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '#lib/api.ts';
	import { app } from '#lib/state.svelte.ts';
	import type { LogEntry, Workout } from '#lib/types.ts';
	import { compile, estimateDuration, type Step } from '#lib/timer/compile.ts';
	import { PlayerEngine } from '#lib/timer/engine.svelte.ts';
	import { beeps, speak, stopSpeaking, unlockAudio } from '#lib/audio/audio.ts';
	import { keepAwake, releaseAwake } from '#lib/wakelock.ts';
	import { ClipRecorder, canRecord, type Facing } from '#lib/camera/camera.ts';
	import { uploads } from '#lib/camera/uploads.svelte.ts';
	import { PROFILES } from '#lib/pose/profiles.ts';
	import type { RepResult } from '#lib/pose/analyzer.ts';
	import { formatClock, formatDuration, formatSide, formatWeight, summarizeLog } from '#lib/format.ts';
	import CameraStage from '#lib/components/CameraStage.svelte';
	import {
		X,
		Play,
		Pause,
		SkipForward,
		SkipBack,
		Check,
		Minus,
		Plus,
		Camera,
		CameraOff,
		SwitchCamera,
		ScanEye,
		Volume2,
		VolumeX,
		CircleAlert,
		CircleCheck,
		Trophy,
		MessageSquare
	} from '@lucide/svelte';

	const id = Number(page.params.id);

	let workout = $state<Workout | null>(null);
	let loadError = $state<string | null>(null);
	let steps = $state<Step[]>([]);
	let engine = $state<PlayerEngine | null>(null);
	let phase = $state<'ready' | 'playing' | 'done'>('ready');

	// Session
	let sessionId = $state<number | null>(null);
	let sessionPromise: Promise<number | null> | null = null;
	let startedAt = '';
	let log = $state<LogEntry[]>([]);
	let notes = $state('');
	let notesSaved = $state(false);

	// Optionen
	let sound = $state(app.settings.sound);
	let voice = $state(app.settings.voice);
	let speakCues = $state(false);
	let cameraOn = $state(false);
	let facing = $state<Facing>('user');
	let showPose = $state(true);
	let stream = $state<MediaStream | null>(null);
	let stage: CameraStage | undefined = $state();
	let cameraError = $state<string | null>(null);

	// Schritt-Zustand
	let repsInput = $state(0);
	let amrapRounds = $state(0);
	let poseReps = $state(0);
	let lastResult = $state<RepResult | null>(null);
	let trackable = $state(true);

	// Aufnahme
	let recorder: ClipRecorder | null = null;
	let recAuto = false;
	let recording = $state(false);
	let recStartedAt = $state(0);
	let recMeta: { exerciseId: number | null; label: string } = { exerciseId: null, label: '' };
	let now = $state(performance.now());

	const step = $derived(engine?.step);
	const exercise = $derived(step?.exerciseId ? app.exerciseMap.get(step.exerciseId) : undefined);
	const profileId = $derived(step?.kind === 'work' && exercise?.poseProfile ? exercise.poseProfile : null);
	const poseActive = $derived(cameraOn && showPose && !!profileId);
	const hasPoseExercises = $derived(steps.some((s) => s.exerciseId && app.exerciseMap.get(s.exerciseId)?.poseProfile));
	const hasAutoRecord = $derived(steps.some((s) => s.record));
	const totalEstimate = $derived(estimateDuration(steps, app.settings.secondsPerRep));
	const summary = $derived(summarizeLog(log));

	onMount(async () => {
		try {
			workout = await api.workout(id);
			steps = compile(workout.structure, app.exerciseMap);
		} catch (err) {
			loadError = (err as Error).message;
		}
	});

	// ---------- Ansagen ----------

	function weightWords(w: Step['weight']): string {
		if (typeof w === 'number') return `${String(w).replace('.', ',')} Kilo`;
		return '';
	}

	function describeForSpeech(s: Step): string {
		if (s.amrapItems) {
			return `AMRAP, ${Math.round((s.durationSec ?? 0) / 60)} Minuten. ${s.amrapItems.map((i) => `${i.reps ?? ''} ${i.name}`).join(', ')}`;
		}
		const parts = [s.exerciseName];
		if (s.side) parts.push(s.side === 'L' ? 'linke Seite' : 'rechte Seite');
		const w = weightWords(s.weight);
		if (w) parts.push(w);
		if (s.targetReps) parts.push(`${s.targetReps} Wiederholungen`);
		else if (s.durationSec && s.kind === 'work') parts.push(`${s.durationSec} Sekunden`);
		return parts.join(', ');
	}

	function announce(s: Step) {
		if (s.kind === 'work') {
			if (sound) beeps.go();
			if (voice) speak(describeForSpeech(s));
		} else if (s.kind === 'rest') {
			if (sound) beeps.rest();
			const next = engine?.nextWork;
			if (voice && (s.durationSec ?? 0) >= 8) {
				speak(s.note === 'Seitenwechsel' ? 'Seite wechseln' : `Pause. Als nächstes: ${next ? describeForSpeech(next) : 'geschafft'}`);
			}
		} else if (s.kind === 'prepare') {
			const next = engine?.nextWork;
			if (voice) speak(`Mach dich bereit. ${next ? `Zuerst: ${describeForSpeech(next)}` : ''}`);
		}
	}

	// ---------- Engine ----------

	function createEngine(): PlayerEngine {
		return new PlayerEngine(steps, {
			onStepStart(s, { catchUp }) {
				repsInput = s.targetReps ?? 0;
				amrapRounds = 0;
				poseReps = 0;
				lastResult = null;
				if (!catchUp) announce(s);
				if (s.kind === 'work' && s.record && cameraOn && stream && !recording) {
					startRecording(true, s);
				}
			},
			onStepEnd(s, info) {
				const flushed = stage?.flush();
				if (flushed) lastResult = flushed;
				if (s.kind === 'work') logStep(s, info.elapsedSec, info.skipped);
				if (recording && recAuto) void stopRecording();
			},
			onSecond(sec, s) {
				if (sound && sec >= 1 && sec <= 3) beeps.tick();
				const dur = s.durationSec ?? 0;
				if (voice && s.kind === 'rest' && dur >= 30 && sec === 10) speak('Noch 10 Sekunden', { interrupt: false });
				if (voice && s.kind === 'work' && dur >= 180 && sec === 60) speak('Noch eine Minute', { interrupt: false });
			},
			onFinish() {
				void finish();
			}
		});
	}

	function logStep(s: Step, elapsedSec: number, skipped: boolean) {
		const at = new Date().toISOString();
		if (s.amrapItems) {
			s.amrapItems.forEach((it, i) => {
				log.push({
					step: s.index,
					blockIndex: s.blockIndex,
					exerciseId: it.exerciseId,
					exerciseName: it.name,
					weight: it.weight,
					reps: it.reps ? it.reps * amrapRounds * (it.unilateral ? 2 : 1) : null,
					rounds: i === 0 ? amrapRounds : undefined,
					durationSec: i === 0 ? Math.round(elapsedSec) : 0,
					skipped,
					at
				});
			});
		} else {
			let reps: number | null;
			if (s.advance === 'manual') reps = repsInput;
			else if (poseReps > 0) reps = poseReps;
			else reps = s.targetReps;
			log.push({
				step: s.index,
				blockIndex: s.blockIndex,
				exerciseId: s.exerciseId,
				exerciseName: s.exerciseName,
				weight: s.weight,
				side: s.side,
				reps: skipped ? null : reps,
				durationSec: Math.round(elapsedSec),
				skipped,
				at
			});
		}
		void persistLog();
	}

	function ensureSession(): Promise<number | null> {
		if (sessionId) return Promise.resolve(sessionId);
		if (!sessionPromise && workout) {
			sessionPromise = api
				.createSession({ workoutId: workout.id, workoutName: workout.name, snapshot: workout.structure, startedAt })
				.then((s) => (sessionId = s.id))
				.catch(() => {
					sessionPromise = null; // später erneut versuchen
					return null;
				});
		}
		return sessionPromise ?? Promise.resolve(null);
	}

	let persistTimer: ReturnType<typeof setTimeout> | undefined;
	function persistLog() {
		clearTimeout(persistTimer);
		persistTimer = setTimeout(async () => {
			const sid = await ensureSession();
			if (sid) await api.patchSession(sid, { log: $state.snapshot(log) as LogEntry[] }).catch(() => {});
		}, 500);
	}

	let ticker: ReturnType<typeof setInterval> | undefined;
	let raf = 0;
	function loop() {
		engine?.tick();
		now = performance.now();
		raf = requestAnimationFrame(loop);
	}

	function onVisibility() {
		engine?.tick();
	}

	async function start() {
		unlockAudio();
		void keepAwake();
		startedAt = new Date().toISOString();
		void ensureSession();
		engine = createEngine();
		phase = 'playing';
		engine.start();
		loop();
		// Fallback-Ticker, falls rAF im Hintergrund pausiert.
		ticker = setInterval(() => engine?.tick(), 250);
		document.addEventListener('visibilitychange', onVisibility);
	}

	async function finish() {
		phase = 'done';
		cancelAnimationFrame(raf);
		clearInterval(ticker);
		if (recording) await stopRecording();
		if (sound) beeps.done();
		if (voice) speak('Geschafft! Starke Leistung.');
		void releaseAwake();
		const sid = await ensureSession();
		if (sid) {
			clearTimeout(persistTimer);
			await api.patchSession(sid, { finishedAt: new Date().toISOString(), log: $state.snapshot(log) as LogEntry[] }).catch(() => {});
		}
	}

	function quit() {
		if (phase !== 'playing') {
			void goto('/');
			return;
		}
		engine?.pause();
		if (confirm('Training beenden? Der bisherige Fortschritt wird gespeichert.')) {
			// Laufenden Arbeitsschritt anteilig loggen.
			const s = engine?.step;
			if (s && s.kind === 'work' && engine) logStep(s, engine.stepElapsedMs / 1000, s.advance === 'timer');
			void finish();
		} else {
			engine?.resume();
		}
	}

	function done() {
		engine?.complete();
	}

	async function saveNotes() {
		const sid = await ensureSession();
		if (sid) {
			await api.patchSession(sid, { notes });
			notesSaved = true;
		}
	}

	// Automatisch weiter, wenn der Pose-Zähler das Ziel erreicht (Einstellung).
	$effect(() => {
		if (!app.settings.autoAdvanceReps || !poseActive || !step || step.advance !== 'manual' || !step.targetReps) return;
		if (poseReps >= step.targetReps) {
			repsInput = poseReps;
			const t = setTimeout(() => engine?.complete(), 900);
			return () => clearTimeout(t);
		}
	});

	// Gezählte Reps übernehmen, solange der Nutzer nicht manuell korrigiert hat.
	function onRep(count: number) {
		const prev = poseReps;
		poseReps = count;
		if (count > 0 && step?.advance === 'manual' && (repsInput === (step.targetReps ?? 0) || repsInput === prev)) {
			repsInput = count;
		}
		if (count > 0 && sound) beeps.rep();
	}

	function onResult(r: RepResult) {
		lastResult = r;
		const warn = r.cues.find((c) => c.level === 'warn');
		if (warn && speakCues) speak(warn.text, { interrupt: false });
	}

	// ---------- Aufnahme ----------

	function startRecording(auto: boolean, s: Step | undefined = step) {
		if (!stream || recording || !canRecord()) return;
		try {
			recorder = new ClipRecorder(stream);
			recorder.start();
		} catch (err) {
			cameraError = `Aufnahme nicht möglich: ${(err as Error).message}`;
			return;
		}
		recAuto = auto;
		recording = true;
		recStartedAt = performance.now();
		const side = s?.side ? ` (${formatSide(s.side)})` : '';
		recMeta = {
			exerciseId: s?.exerciseId ?? null,
			label: s && s.kind === 'work' ? `${s.exerciseName}${side} · ${s.progress}`.trim() : `Aufnahme ${new Date().toLocaleTimeString('de-DE')}`
		};
	}

	async function stopRecording() {
		if (!recorder) return;
		const r = recorder;
		// Metadaten sofort sichern – eine neue Aufnahme kann recMeta überschreiben, bevor stop() fertig ist.
		const meta = recMeta;
		recorder = null;
		recording = false;
		try {
			const clip = await r.stop();
			if (clip.blob.size > 0) {
				const sid = await ensureSession();
				uploads.enqueue(clip, { sessionId: sid, exerciseId: meta.exerciseId, label: meta.label });
			}
		} catch (err) {
			cameraError = `Aufnahme fehlgeschlagen: ${(err as Error).message}`;
		}
	}

	function toggleRecording() {
		if (recording) void stopRecording();
		else startRecording(false);
	}

	function toggleCamera() {
		if (cameraOn && recording) void stopRecording();
		cameraOn = !cameraOn;
		cameraError = null;
	}

	onDestroy(() => {
		cancelAnimationFrame(raf);
		clearInterval(ticker);
		clearTimeout(persistTimer);
		document.removeEventListener('visibilitychange', onVisibility);
		if (recording) void stopRecording();
		stopSpeaking();
		void releaseAwake();
		// Abgebrochen ohne „Beenden“ (z. B. Zurück-Geste): Stand speichern.
		if (phase === 'playing' && sessionId) {
			void api.patchSession(sessionId, { finishedAt: new Date().toISOString(), log: $state.snapshot(log) as LogEntry[] }).catch(() => {});
		}
	});

	// Ringfortschritt
	const R = 46;
	const C = 2 * Math.PI * R;
</script>

<svelte:head><title>{workout?.name ?? 'Training'} · Workout</title></svelte:head>

{#if loadError}
	<div class="grid min-h-dvh place-items-center p-6 text-center">
		<div class="space-y-4">
			<p class="text-danger">{loadError}</p>
			<a href="/" class="btn-secondary">Zurück</a>
		</div>
	</div>
{:else if !workout}
	<div class="grid min-h-dvh place-items-center">
		<div class="size-10 animate-spin rounded-full border-4 border-line border-t-accent"></div>
	</div>
{:else if phase === 'ready'}
	<!-- ---------- Startbildschirm ---------- -->
	<div class="mx-auto flex min-h-dvh max-w-lg flex-col px-4 pt-[max(1rem,env(safe-area-inset-top))] pb-[max(1rem,env(safe-area-inset-bottom))]">
		<div class="flex items-center">
			<button class="icon-btn" onclick={() => goto('/')} aria-label="Zurück"><X class="size-6" /></button>
		</div>
		<div class="flex flex-1 flex-col justify-center gap-6 py-6">
			<div>
				<p class="text-sm font-semibold tracking-wide text-accent uppercase">Bereit?</p>
				<h1 class="text-3xl font-bold tracking-tight">{workout.name}</h1>
				<p class="mt-1 text-muted">
					~{formatDuration(totalEstimate)} · {steps.filter((s) => s.kind === 'work').length} Arbeitsphasen
				</p>
				{#if workout.notes}<p class="mt-3 text-sm text-muted">{workout.notes}</p>{/if}
			</div>

			<div class="card divide-y divide-line">
				<label class="flex min-h-14 items-center gap-3 px-4">
					<Volume2 class="size-5 text-muted" /><span class="flex-1 font-medium">Töne</span>
					<input type="checkbox" class="size-5 accent-[var(--accent)]" bind:checked={sound} />
				</label>
				<label class="flex min-h-14 items-center gap-3 px-4">
					<MessageSquare class="size-5 text-muted" /><span class="flex-1 font-medium">Sprachansagen</span>
					<input type="checkbox" class="size-5 accent-[var(--accent)]" bind:checked={voice} />
				</label>
				<label class="flex min-h-14 items-center gap-3 px-4">
					<Camera class="size-5 text-muted" />
					<span class="flex-1">
						<span class="block font-medium">Kamera-Spiegel</span>
						{#if hasAutoRecord}<span class="block text-xs text-muted">Einige Blöcke werden automatisch aufgenommen</span>{/if}
					</span>
					<input type="checkbox" class="size-5 accent-[var(--accent)]" bind:checked={cameraOn} />
				</label>
				{#if cameraOn}
					<label class="flex min-h-14 items-center gap-3 px-4">
						<ScanEye class="size-5 text-muted" />
						<span class="flex-1">
							<span class="block font-medium">Formerkennung</span>
							<span class="block text-xs text-muted">Skelett, Wiederholungen zählen, Hinweise</span>
						</span>
						<input type="checkbox" class="size-5 accent-[var(--accent)]" bind:checked={showPose} />
					</label>
					{#if showPose}
						<label class="flex min-h-14 items-center gap-3 px-4">
							<MessageSquare class="size-5 text-muted" /><span class="flex-1 font-medium">Form-Hinweise ansagen</span>
							<input type="checkbox" class="size-5 accent-[var(--accent)]" bind:checked={speakCues} />
						</label>
					{/if}
				{/if}
			</div>

			{#if cameraOn && hasPoseExercises && showPose}
				<p class="rounded-xl bg-accent-soft px-4 py-3 text-sm text-fg">
					<strong>Tipp:</strong> Für Swings, Squats und Kreuzheben das Handy <strong>seitlich</strong> in ~2–3 m Abstand aufstellen, beim Drücken
					<strong>frontal</strong>. Der ganze Körper sollte im Bild sein.
				</p>
			{/if}
		</div>
		<button class="btn-primary h-16 text-lg" onclick={start} disabled={steps.length === 0}>
			<Play class="size-6 fill-current" /> Los geht’s
		</button>
	</div>
{:else if phase === 'playing' && engine && step}
	<!-- ---------- Player ---------- -->
	{@const rem = engine.remainingSec}
	{@const isWork = step.kind === 'work'}
	{@const isRest = step.kind !== 'work'}
	{@const nextWork = engine.nextWork}
	<div
		class="fixed inset-0 flex flex-col overflow-hidden select-none {cameraOn ? 'bg-black text-white' : isRest ? 'bg-rest-soft' : 'bg-bg'}"
		style={cameraOn ? '' : `background-color: color-mix(in srgb, var(${isRest ? '--rest' : '--accent'}) ${isRest ? 10 : 6}%, var(--bg));`}
	>
		{#if cameraOn}
			<CameraStage
				bind:this={stage}
				bind:stream
				{facing}
				mirror={facing === 'user'}
				showPose={showPose}
				profile={profileId}
				resetKey={engine.index}
				onrep={onRep}
				onresult={onResult}
				onerror={(m) => (cameraError = m)}
				ontrackable={(ok) => (trackable = ok)}
			/>
			<div class="pointer-events-none absolute inset-x-0 top-0 h-36 bg-gradient-to-b from-black/70 to-transparent"></div>
			<div class="pointer-events-none absolute inset-x-0 bottom-0 h-2/3 bg-gradient-to-t from-black/85 via-black/40 to-transparent"></div>
		{/if}

		<!-- Gesamtfortschritt -->
		<div class="relative z-10 h-1 w-full {cameraOn ? 'bg-white/20' : 'bg-line'}">
			<div class="h-full bg-accent transition-[width] duration-300" style="width: {engine.overallProgress * 100}%"></div>
		</div>

		<!-- Kopfzeile -->
		<header class="relative z-10 flex items-center gap-1 px-2 pt-[max(0.25rem,env(safe-area-inset-top))]">
			<button class="icon-btn {cameraOn ? 'text-white hover:bg-white/10' : ''}" onclick={quit} aria-label="Beenden"><X class="size-6" /></button>
			<div class="min-w-0 flex-1 px-1 leading-tight">
				<p class="truncate text-sm font-semibold">{step.blockLabel}{step.progress ? ` · ${step.progress}` : ''}</p>
				<p class="tabular text-xs opacity-70">{formatClock(engine.totalElapsedMs / 1000)} · Schritt {step.index + 1}/{steps.length}</p>
			</div>
			<button class="icon-btn {cameraOn ? 'text-white hover:bg-white/10' : ''}" onclick={() => (sound = !sound)} aria-label="Ton">
				{#if sound}<Volume2 class="size-5" />{:else}<VolumeX class="size-5" />{/if}
			</button>
			{#if cameraOn}
				<button class="icon-btn text-white hover:bg-white/10" onclick={() => (facing = facing === 'user' ? 'environment' : 'user')} aria-label="Kamera wechseln">
					<SwitchCamera class="size-5" />
				</button>
				<button class="icon-btn hover:bg-white/10 {showPose ? 'text-accent' : 'text-white'}" onclick={() => (showPose = !showPose)} aria-label="Formerkennung">
					<ScanEye class="size-5" />
				</button>
			{/if}
			<button class="icon-btn {cameraOn ? 'text-white hover:bg-white/10' : ''}" onclick={toggleCamera} aria-label="Kamera">
				{#if cameraOn}<CameraOff class="size-5" />{:else}<Camera class="size-5" />{/if}
			</button>
		</header>

		{#if cameraOn}
			<div class="relative z-10 flex flex-wrap items-center gap-2 px-3 pt-1">
				{#if recording}
					<span class="flex items-center gap-1.5 rounded-full bg-red-600 px-3 py-1 text-xs font-bold">
						<span class="size-2 animate-pulse rounded-full bg-white"></span>
						REC {formatClock((now - recStartedAt) / 1000)}
					</span>
				{/if}
				{#if poseActive && !trackable}
					<span class="rounded-full bg-black/60 px-3 py-1 text-xs">Ganzer Körper nicht sichtbar · {PROFILES[profileId!].cameraHint}</span>
				{/if}
				{#if cameraError}
					<span class="flex items-center gap-1.5 rounded-full bg-red-600/90 px-3 py-1 text-xs"><CircleAlert class="size-3.5" />{cameraError}</span>
				{/if}
			</div>
		{/if}

		<!-- Hauptbereich -->
		<main
			class="relative z-10 flex flex-1 flex-col items-center justify-center px-4 text-center {cameraOn
				? 'justify-end pb-2 landscape:items-start landscape:pl-6 landscape:text-left'
				: ''}"
		>
			{#if step.amrapItems}
				<div class="w-full max-w-md space-y-2 {cameraOn ? 'rounded-2xl bg-black/40 p-3 backdrop-blur-sm' : ''}">
					<p class="text-sm font-bold tracking-widest text-accent uppercase">AMRAP</p>
					{#each step.amrapItems as it (it.exerciseId)}
						<div class="flex items-center justify-between rounded-xl px-3 py-2 text-left {cameraOn ? 'bg-white/10' : 'bg-surface'}">
							<span class="font-semibold">{it.reps ? `${it.reps}× ` : ''}{it.name}{it.unilateral ? ' (je Seite)' : ''}</span>
							<span class="text-sm opacity-70">{formatWeight(it.weight)}</span>
						</div>
					{/each}
				</div>
			{:else if isWork}
				<div class="flex flex-wrap items-center justify-center gap-2 {cameraOn ? 'landscape:justify-start' : ''}">
					{#if step.weight !== null}
						<span class="rounded-full px-3 py-1 text-sm font-bold {cameraOn ? 'bg-white/15' : 'bg-accent-soft text-accent'}">{formatWeight(step.weight)}</span>
					{/if}
					{#if step.side}
						<span class="rounded-full px-3 py-1 text-sm font-bold {cameraOn ? 'bg-white/15' : 'bg-surface-2'}">{formatSide(step.side)}</span>
					{/if}
				</div>
				<h1 class="mt-2 leading-tight font-extrabold tracking-tight text-balance {cameraOn ? 'text-2xl drop-shadow-lg sm:text-3xl landscape:text-xl' : 'text-4xl sm:text-5xl'}">{step.exerciseName}</h1>
			{:else}
				<p class="text-sm font-bold tracking-widest uppercase {cameraOn ? 'text-white/80' : 'text-rest'}">
					{step.kind === 'prepare' ? 'Vorbereitung' : (step.note ?? 'Pause')}
				</p>
			{/if}

			<!-- Timer / Reps -->
			{#if rem !== null}
				{#if cameraOn}
					<p class="tabular mt-1 text-6xl font-black tracking-tight drop-shadow-lg landscape:text-5xl {rem <= 3.5 ? 'text-accent' : ''}">{formatClock(Math.ceil(rem))}</p>
				{:else}
					<div class="relative mt-4 aspect-square w-[min(72vw,42vh,22rem)]">
						<svg viewBox="0 0 100 100" class="size-full -rotate-90">
							<circle cx="50" cy="50" r={R} fill="none" stroke="currentColor" stroke-width="4" class="text-line" />
							<circle
								cx="50"
								cy="50"
								r={R}
								fill="none"
								stroke="currentColor"
								stroke-width="4"
								stroke-linecap="round"
								class={isRest ? 'text-rest' : 'text-accent'}
								stroke-dasharray={C}
								stroke-dashoffset={C * engine.stepProgress}
							/>
						</svg>
						<div class="absolute inset-0 flex flex-col items-center justify-center">
							<span class="tabular text-[min(20vw,12vh,6.5rem)] leading-none font-black tracking-tight {rem <= 3.5 ? 'text-accent' : ''}">
								{formatClock(Math.ceil(rem))}
							</span>
							{#if isWork && step.targetReps}
								<span class="mt-2 text-lg font-semibold text-muted">{step.targetReps} Wdh.</span>
							{/if}
						</div>
					</div>
				{/if}
			{:else if isWork && step.advance === 'manual'}
				<div class="mt-4 flex flex-col items-center">
					{#if poseActive}
						<p class="tabular leading-none font-black {cameraOn ? 'text-6xl drop-shadow-lg landscape:text-5xl' : 'text-[min(26vw,14vh,8rem)]'}">
							{poseReps}<span class="text-[0.45em] opacity-60">/{step.targetReps}</span>
						</p>
						<p class="text-sm opacity-70">erkannte Wiederholungen</p>
					{:else}
						<p class="tabular leading-none font-black {cameraOn ? 'text-6xl drop-shadow-lg landscape:text-5xl' : 'text-[min(26vw,14vh,8rem)]'}">{step.targetReps}</p>
						<p class="text-lg font-semibold opacity-70">Wiederholungen{step.side ? ` · ${formatSide(step.side)}` : ''}</p>
					{/if}
					<p class="tabular mt-2 text-sm opacity-60">{formatClock(engine.stepElapsedMs / 1000)}</p>
				</div>
			{/if}

			{#if isWork && poseActive && rem !== null && poseReps > 0}
				<p class="tabular mt-1 text-2xl font-bold">{poseReps} Wdh. erkannt</p>
			{/if}

			{#if isRest && nextWork}
				<div class="mt-6 rounded-2xl px-5 py-3 {cameraOn ? 'bg-black/40 backdrop-blur-sm' : 'bg-surface shadow-card'}">
					<p class="text-xs font-semibold tracking-wide uppercase opacity-60">Als nächstes</p>
					<p class="text-lg font-bold">{nextWork.amrapItems ? 'AMRAP' : nextWork.exerciseName}</p>
					<p class="text-sm opacity-70">
						{[nextWork.targetReps ? `${nextWork.targetReps} Wdh.` : nextWork.durationSec ? `${nextWork.durationSec} s` : '', formatWeight(nextWork.weight), formatSide(nextWork.side)]
							.filter(Boolean)
							.join(' · ')}
					</p>
				</div>
			{/if}

			{#if lastResult && poseActive}
				<div class="mt-4 flex flex-wrap justify-center gap-2">
					{#each lastResult.cues as cue (cue.id)}
						<span
							class="flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-semibold {cue.level === 'ok'
								? 'bg-emerald-500/90 text-white'
								: 'bg-amber-400/95 text-black'}"
						>
							{#if cue.level === 'ok'}<CircleCheck class="size-4" />{:else}<CircleAlert class="size-4" />{/if}
							{cue.text}
						</span>
					{/each}
				</div>
			{/if}
		</main>

		<!-- Steuerung -->
		<footer class="relative z-10 space-y-3 px-4 pt-2 pb-[max(1rem,env(safe-area-inset-bottom))]">
			{#if isWork && step.advance === 'manual'}
				<div class="mx-auto flex max-w-sm items-center justify-center gap-3">
					<button class="icon-btn size-14 rounded-full {cameraOn ? 'bg-white/15 text-white' : 'bg-surface shadow-card'}" onclick={() => (repsInput = Math.max(0, repsInput - 1))} aria-label="Weniger">
						<Minus class="size-6" />
					</button>
					<div class="w-28 text-center">
						<p class="tabular text-3xl font-black">{repsInput}</p>
						<p class="text-xs opacity-60">geschafft</p>
					</div>
					<button class="icon-btn size-14 rounded-full {cameraOn ? 'bg-white/15 text-white' : 'bg-surface shadow-card'}" onclick={() => (repsInput += 1)} aria-label="Mehr">
						<Plus class="size-6" />
					</button>
				</div>
			{/if}
			{#if step.amrapItems}
				<div class="mx-auto flex max-w-sm items-center justify-center gap-3">
					<button class="icon-btn size-14 rounded-full {cameraOn ? 'bg-white/15 text-white' : 'bg-surface shadow-card'}" onclick={() => (amrapRounds = Math.max(0, amrapRounds - 1))} aria-label="Runde zurück">
						<Minus class="size-6" />
					</button>
					<button class="btn h-16 flex-1 rounded-2xl text-xl {cameraOn ? 'bg-white/15 text-white' : 'bg-surface shadow-card'}" onclick={() => (amrapRounds += 1)}>
						Runde <span class="tabular font-black">{amrapRounds}</span> <Plus class="size-5" />
					</button>
				</div>
			{/if}

			<div class="mx-auto flex max-w-md items-center gap-3">
				<button class="icon-btn size-14 shrink-0 rounded-full {cameraOn ? 'bg-white/15 text-white' : 'bg-surface shadow-card'}" onclick={() => engine?.back()} aria-label="Zurück">
					<SkipBack class="size-6" />
				</button>
				<button
					class="icon-btn size-14 shrink-0 rounded-full {cameraOn ? 'bg-white/15 text-white' : 'bg-surface shadow-card'}"
					onclick={() => engine?.togglePause()}
					aria-label={engine.status === 'paused' ? 'Fortsetzen' : 'Pause'}
				>
					{#if engine.status === 'paused'}<Play class="size-6 fill-current" />{:else}<Pause class="size-6 fill-current" />{/if}
				</button>
				{#if isWork && step.advance === 'manual'}
					<button class="btn-primary h-16 flex-1 rounded-2xl text-lg" onclick={done}>
						<Check class="size-6" strokeWidth={3} /> Fertig
					</button>
				{:else}
					<button class="btn h-16 flex-1 rounded-2xl text-lg {cameraOn ? 'bg-white/15 text-white' : 'bg-surface text-fg shadow-card'}" onclick={() => engine?.skip()}>
						{isRest ? 'Überspringen' : 'Weiter'} <SkipForward class="size-5" />
					</button>
				{/if}
				{#if cameraOn && canRecord()}
					<button
						class="grid size-14 shrink-0 place-items-center rounded-full border-4 border-white/80 {recording ? 'bg-white/10' : 'bg-transparent'}"
						onclick={toggleRecording}
						aria-label={recording ? 'Aufnahme stoppen' : 'Aufnahme starten'}
					>
						<span class="block bg-red-600 transition-all {recording ? 'size-5 rounded-md' : 'size-9 rounded-full'}"></span>
					</button>
				{/if}
			</div>
			{#if engine.status === 'paused'}
				<p class="text-center text-sm font-semibold opacity-80">Pausiert</p>
			{/if}
		</footer>
	</div>
{:else if phase === 'done'}
	<!-- ---------- Abschluss ---------- -->
	<div class="mx-auto flex min-h-dvh max-w-lg flex-col px-4 pt-[max(1.5rem,env(safe-area-inset-top))] pb-[max(1rem,env(safe-area-inset-bottom))]">
		<div class="flex flex-1 flex-col justify-center gap-6">
			<div class="text-center">
				<div class="mx-auto mb-4 grid size-20 place-items-center rounded-full bg-accent-soft text-accent">
					<Trophy class="size-10" />
				</div>
				<h1 class="text-3xl font-bold tracking-tight">Geschafft!</h1>
				<p class="text-muted">{workout.name}</p>
			</div>
			<div class="grid grid-cols-3 gap-2 text-center">
				<div class="card p-3">
					<p class="tabular text-2xl font-black">{formatClock((engine?.totalElapsedMs ?? 0) / 1000)}</p>
					<p class="text-xs text-muted">Dauer</p>
				</div>
				<div class="card p-3">
					<p class="tabular text-2xl font-black">{summary.reps}</p>
					<p class="text-xs text-muted">Wiederholungen</p>
				</div>
				<div class="card p-3">
					<p class="tabular text-2xl font-black">{Math.round(summary.volumeKg).toLocaleString('de-DE')}</p>
					<p class="text-xs text-muted">kg bewegt</p>
				</div>
			</div>
			<div>
				<label class="label" for="notes">Wie lief’s? (optional)</label>
				<textarea id="notes" class="input min-h-24 py-2" bind:value={notes} oninput={() => (notesSaved = false)} placeholder="z. B. Swings mit 16 kg fühlten sich leicht an …"></textarea>
				{#if notes}
					<button class="btn-secondary mt-2 w-full" onclick={saveNotes} disabled={notesSaved}>
						{notesSaved ? 'Gespeichert ✓' : 'Notiz speichern'}
					</button>
				{/if}
			</div>
			{#if uploads.pending > 0}
				<p class="text-center text-sm text-muted">Aufnahmen werden noch hochgeladen – bitte die Seite offen lassen.</p>
			{/if}
		</div>
		<div class="grid gap-2">
			{#if sessionId}<a href="/history/{sessionId}" class="btn-primary">Zur Auswertung</a>{/if}
			<a href="/" class="btn-secondary">Fertig</a>
		</div>
	</div>
{/if}

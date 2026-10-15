<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { PoseLandmarker } from '@mediapipe/tasks-vision';
	import { api } from '#lib/api.ts';
	import type { PoseProfileId, Recording } from '#lib/types.ts';
	import { loadLandmarker, detect } from '#lib/pose/landmarker.ts';
	import { drawSkeleton, clearCanvas } from '#lib/pose/draw.ts';
	import { PoseAnalyzer, type RepResult } from '#lib/pose/analyzer.ts';
	import { PROFILES, PROFILE_OPTIONS } from '#lib/pose/profiles.ts';
	import type { Metrics } from '#lib/pose/geometry.ts';
	import { formatBytes, formatClock, formatDateTime } from '#lib/format.ts';
	import { ChevronLeft, Play, Pause, StepBack, StepForward, ScanEye, Trash, Download, CircleCheck, CircleAlert } from '@lucide/svelte';

	const id = Number(page.params.id);
	let rec = $state<Recording | null>(null);
	let error = $state<string | null>(null);

	let video: HTMLVideoElement | undefined = $state();
	let canvas: HTMLCanvasElement | undefined = $state();
	let playing = $state(false);
	let current = $state(0);
	let duration = $state(0);
	let rate = $state(1);
	let mirror = $state(false);
	let showPose = $state(true);
	let profile = $state<PoseProfileId | null>(null);
	let poseState = $state<'off' | 'loading' | 'ready' | 'error'>('off');
	let reps = $state(0);
	let results = $state<RepResult[]>([]);
	let metrics = $state<Metrics | null>(null);
	let note = $state('');
	let noteDirty = $state(false);

	let landmarker: PoseLandmarker | null = null;
	let analyzer = new PoseAnalyzer(null);
	let raf = 0;
	let lastTime = -1;

	onMount(async () => {
		try {
			rec = await api.recording(id);
			profile = rec.poseProfile;
			note = rec.note;
			analyzer = new PoseAnalyzer(profile);
		} catch (err) {
			error = (err as Error).message;
		}
	});

	function resetAnalysis() {
		analyzer = new PoseAnalyzer(profile);
		reps = 0;
		results = [];
		lastTime = -1;
	}

	$effect(() => {
		void profile;
		resetAnalysis();
	});

	$effect(() => {
		if (!showPose) {
			cancelAnimationFrame(raf);
			if (canvas) clearCanvas(canvas);
			poseState = 'off';
			return;
		}
		poseState = landmarker ? 'ready' : 'loading';
		let stopped = false;
		loadLandmarker()
			.then((lm) => {
				if (stopped) return;
				landmarker = lm;
				poseState = 'ready';
				loop();
			})
			.catch(() => (poseState = 'error'));
		return () => {
			stopped = true;
			cancelAnimationFrame(raf);
		};
	});

	function loop() {
		raf = requestAnimationFrame(loop);
		if (!video || !canvas || !landmarker) return;
		if (video.currentTime === lastTime) return;
		lastTime = video.currentTime;
		const lms = detect(landmarker, video);
		const out = analyzer.process(lms, video.currentTime * 1000, video.videoWidth / Math.max(1, video.videoHeight));
		metrics = out.metrics;
		if (analyzer.reps !== reps) reps = analyzer.reps;
		if (out.result) results = [...results, out.result];
		drawSkeleton(canvas, video, out.landmarks, {
			color: out.trackable ? 'rgba(255,255,255,0.9)' : 'rgba(255,255,255,0.45)',
			jointColor: getComputedStyle(document.documentElement).getPropertyValue('--accent').trim() || '#fb7b30',
			fit: 'contain'
		});
	}

	function togglePlay() {
		if (!video) return;
		if (video.paused) void video.play();
		else video.pause();
	}

	function stepFrame(dir: number) {
		if (!video) return;
		video.pause();
		video.currentTime = Math.max(0, Math.min(duration, video.currentTime + dir / 30));
	}

	function setRate(r: number) {
		rate = r;
		if (video) video.playbackRate = r;
	}

	function onSeeking() {
		// Rückwärts springen → Zählung neu beginnen
		if (video && video.currentTime < lastTime) resetAnalysis();
	}

	function onEnded() {
		const last = analyzer.flush();
		if (last) results = [...results, last];
	}

	async function saveNote() {
		if (!rec) return;
		rec = await api.patchRecording(rec.id, note);
		noteDirty = false;
	}

	async function remove() {
		if (!rec || !confirm('Aufnahme löschen?')) return;
		await api.deleteRecording(rec.id);
		await goto(rec.sessionId ? `/history/${rec.sessionId}` : '/history');
	}

	onDestroy(() => cancelAnimationFrame(raf));

	const signalLabel = $derived.by(() => {
		if (!metrics || !profile) return null;
		const p = PROFILES[profile];
		return `${p.signalName}: ${Math.round(p.signal(metrics))}°`;
	});
</script>

<svelte:head><title>{rec?.label || 'Aufnahme'} · Workout</title></svelte:head>

<div class="mb-3 flex items-center gap-2">
	<a href={rec?.sessionId ? `/history/${rec.sessionId}` : '/history'} class="icon-btn -ml-3" aria-label="Zurück"><ChevronLeft class="size-6" /></a>
	<div class="min-w-0 flex-1">
		<h1 class="truncate text-lg font-bold">{rec?.label || rec?.exerciseName || 'Aufnahme'}</h1>
		{#if rec}<p class="text-xs text-muted">{formatDateTime(rec.createdAt)} · {formatBytes(rec.sizeBytes)}</p>{/if}
	</div>
	{#if rec}
		<a class="icon-btn" href={api.recordingUrl(rec.id)} download aria-label="Herunterladen"><Download class="size-5" /></a>
		<button class="icon-btn text-danger" onclick={remove} aria-label="Löschen"><Trash class="size-5" /></button>
	{/if}
</div>

{#if error}
	<p class="card p-4 text-danger">{error}</p>
{:else if rec}
	<div class="relative -mx-4 overflow-hidden bg-black md:mx-0 md:rounded-2xl">
		<!-- svelte-ignore a11y_media_has_caption -->
		<video
			bind:this={video}
			src={api.recordingUrl(rec.id)}
			class="mx-auto max-h-[62dvh] w-full object-contain {mirror ? '-scale-x-100' : ''}"
			playsinline
			preload="auto"
			bind:currentTime={current}
			bind:duration
			onplay={() => (playing = true)}
			onpause={() => (playing = false)}
			onseeking={onSeeking}
			onended={onEnded}
			onclick={togglePlay}
		></video>
		<canvas bind:this={canvas} class="pointer-events-none absolute inset-0 size-full {mirror ? '-scale-x-100' : ''}"></canvas>
		{#if poseState === 'loading'}
			<div class="absolute top-3 left-3 rounded-full bg-black/60 px-3 py-1 text-xs text-white">Pose-Erkennung lädt …</div>
		{/if}
		{#if showPose && profile}
			<div class="absolute top-3 right-3 rounded-xl bg-black/60 px-3 py-1.5 text-right text-white">
				<p class="tabular text-2xl leading-none font-black">{reps}</p>
				<p class="text-[10px] opacity-75">Wdh.</p>
			</div>
			{#if signalLabel}
				<div class="tabular absolute bottom-3 left-3 rounded-full bg-black/60 px-3 py-1 text-xs text-white">{signalLabel}</div>
			{/if}
		{/if}
	</div>

	<!-- Steuerung -->
	<div class="mt-3 space-y-3">
		<input
			type="range"
			min="0"
			max={duration || 0}
			step="0.01"
			bind:value={current}
			oninput={() => video && (video.currentTime = current)}
			class="w-full accent-[var(--accent)]"
			aria-label="Position"
		/>
		<div class="flex items-center justify-between gap-2">
			<span class="tabular w-20 text-sm text-muted">{formatClock(current)} / {formatClock(duration || 0)}</span>
			<div class="flex items-center gap-1">
				<button class="icon-btn" onclick={() => stepFrame(-1)} aria-label="Bild zurück"><StepBack class="size-5" /></button>
				<button class="grid size-14 place-items-center rounded-full bg-accent text-accent-fg" onclick={togglePlay} aria-label={playing ? 'Pause' : 'Abspielen'}>
					{#if playing}<Pause class="size-6 fill-current" />{:else}<Play class="size-6 fill-current" />{/if}
				</button>
				<button class="icon-btn" onclick={() => stepFrame(1)} aria-label="Bild vor"><StepForward class="size-5" /></button>
			</div>
			<div class="w-20"></div>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			{#each [0.25, 0.5, 1] as r (r)}
				<button class="chip {rate === r ? 'chip-active' : ''}" onclick={() => setRate(r)}>{r}×</button>
			{/each}
			<span class="flex-1"></span>
			<button class="chip {mirror ? 'chip-active' : ''}" onclick={() => (mirror = !mirror)}>Spiegeln</button>
			<button class="chip {showPose ? 'chip-active' : ''}" onclick={() => (showPose = !showPose)}><ScanEye class="mr-1 size-4" /> Skelett</button>
		</div>

		<div>
			<label class="label" for="profile">Analyse als</label>
			<select id="profile" class="input" bind:value={profile}>
				<option value={null}>Keine Zählung (nur Skelett)</option>
				{#each PROFILE_OPTIONS as o (o.id)}
					<option value={o.id}>{o.label}</option>
				{/each}
			</select>
			{#if profile}<p class="mt-1 text-xs text-muted">{PROFILES[profile].cameraHint}. Zum Analysieren abspielen – am besten in 0,5×.</p>{/if}
		</div>

		{#if results.length > 0}
			<section class="card overflow-hidden">
				<h2 class="border-b border-line px-4 py-2 text-sm font-bold text-muted">Wiederholungen</h2>
				<ul class="divide-y divide-line">
					{#each results as r (r.rep)}
						<li class="flex items-start gap-3 px-4 py-2 text-sm">
							<span class="tabular w-6 font-bold">{r.rep}</span>
							<span class="flex flex-1 flex-col gap-1">
								{#each r.cues as c (c.id)}
									<span class="flex items-center gap-1.5 {c.level === 'ok' ? 'text-ok' : 'text-warn'}">
										{#if c.level === 'ok'}<CircleCheck class="size-4 shrink-0" />{:else}<CircleAlert class="size-4 shrink-0" />{/if}
										{c.text}
									</span>
								{/each}
							</span>
						</li>
					{/each}
				</ul>
			</section>
			<p class="text-xs text-muted">Hinweise basieren auf groben 2D-Heuristiken – als Anhaltspunkt, nicht als exakte Bewertung.</p>
		{/if}

		<div>
			<label class="label" for="note">Notiz</label>
			<textarea id="note" class="input min-h-20 py-2" bind:value={note} oninput={() => (noteDirty = true)} placeholder="z. B. Hüfte oben nicht ganz gestreckt"></textarea>
			{#if noteDirty}<button class="btn-primary mt-2 w-full" onclick={saveNote}>Notiz speichern</button>{/if}
		</div>
	</div>
{:else}
	<div class="aspect-video animate-pulse rounded-2xl bg-surface-2"></div>
{/if}

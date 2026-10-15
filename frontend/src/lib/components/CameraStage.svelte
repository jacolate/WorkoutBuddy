<script lang="ts">
	// Kamerabild (als Spiegel) mit optionalem Pose-Skelett und Rep-Zählung.
	import { onDestroy, untrack } from 'svelte';
	import type { PoseLandmarker } from '@mediapipe/tasks-vision';
	import { openCamera, stopStream, CameraError, type Facing } from '#lib/camera/camera.ts';
	import { loadLandmarker, detect } from '#lib/pose/landmarker.ts';
	import { drawSkeleton, clearCanvas } from '#lib/pose/draw.ts';
	import { PoseAnalyzer, type RepResult } from '#lib/pose/analyzer.ts';
	import type { PoseProfileId } from '#lib/types.ts';

	let {
		facing = 'user',
		mirror = true,
		showPose = false,
		profile = null,
		resetKey = 0,
		stream = $bindable(null),
		onrep,
		onresult,
		onerror,
		ontrackable
	}: {
		facing?: Facing;
		mirror?: boolean;
		showPose?: boolean;
		profile?: PoseProfileId | null;
		/** Ändert sich der Wert, startet der Rep-Zähler neu (z. B. neuer Satz). */
		resetKey?: number | string;
		stream?: MediaStream | null;
		onrep?: (count: number) => void;
		onresult?: (r: RepResult) => void;
		onerror?: (msg: string) => void;
		ontrackable?: (ok: boolean) => void;
	} = $props();

	let video: HTMLVideoElement | undefined = $state();
	let canvas: HTMLCanvasElement | undefined = $state();
	let poseState = $state<'off' | 'loading' | 'ready' | 'error'>('off');
	let landmarker: PoseLandmarker | null = null;
	let analyzer: PoseAnalyzer | null = null;
	let raf = 0;
	let lastVideoTime = -1;
	let lastTrackable: boolean | null = null;
	let destroyed = false;
	let accent = '';

	// Kamera öffnen / bei Wechsel der Kamera neu öffnen.
	$effect(() => {
		const f = facing;
		let cancelled = false;
		let opened: MediaStream | null = null;
		openCamera(f)
			.then((s) => {
				if (cancelled || destroyed) {
					stopStream(s);
					return;
				}
				opened = s;
				stream = s;
			})
			.catch((err) => onerror?.(err instanceof CameraError ? err.message : String(err)));
		return () => {
			cancelled = true;
			stopStream(opened);
			if (stream === opened) stream = null;
		};
	});

	$effect(() => {
		if (video && stream && video.srcObject !== stream) {
			video.srcObject = stream;
			video.play().catch(() => {});
		}
	});

	// Analyzer neu anlegen, wenn Profil oder Schritt wechselt.
	$effect(() => {
		void resetKey;
		analyzer = new PoseAnalyzer(profile);
		untrack(() => onrep?.(0));
	});

	// Pose-Modell bei Bedarf laden und Schleife starten.
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
			.catch((err) => {
				console.error(err);
				poseState = 'error';
				onerror?.('Pose-Erkennung konnte nicht geladen werden.');
			});
		return () => {
			stopped = true;
			cancelAnimationFrame(raf);
		};
	});

	function loop() {
		raf = requestAnimationFrame(loop);
		if (!video || !canvas || !landmarker || !analyzer) return;
		if (video.currentTime === lastVideoTime) return;
		lastVideoTime = video.currentTime;
		let lms;
		try {
			lms = detect(landmarker, video);
		} catch (err) {
			console.warn(err);
			return;
		}
		const aspect = video.videoWidth / Math.max(1, video.videoHeight);
		const before = analyzer.reps;
		const out = analyzer.process(lms, performance.now(), aspect);
		if (analyzer.reps !== before) onrep?.(analyzer.reps);
		if (out.result) onresult?.(out.result);
		const trackable = !!lms && out.trackable;
		if (trackable !== lastTrackable) {
			lastTrackable = trackable;
			ontrackable?.(trackable);
		}
		accent ||= getComputedStyle(document.documentElement).getPropertyValue('--accent').trim() || '#fb7b30';
		drawSkeleton(canvas, video, out.landmarks, {
			color: trackable ? 'rgba(255,255,255,0.85)' : 'rgba(255,255,255,0.4)',
			jointColor: accent
		});
	}

	/** Offene Wiederholung abschließen (vor einem Schrittwechsel). */
	export function flush(): RepResult | undefined {
		return analyzer?.flush();
	}

	onDestroy(() => {
		destroyed = true;
		cancelAnimationFrame(raf);
		stopStream(stream);
	});
</script>

<div class="absolute inset-0 overflow-hidden bg-black">
	<!-- svelte-ignore a11y_media_has_caption -->
	<video bind:this={video} class="size-full object-cover {mirror ? '-scale-x-100' : ''}" playsinline muted autoplay></video>
	<canvas bind:this={canvas} class="pointer-events-none absolute inset-0 size-full {mirror ? '-scale-x-100' : ''}"></canvas>
	{#if poseState === 'loading'}
		<div class="absolute top-1/2 left-1/2 -translate-1/2 rounded-full bg-black/60 px-4 py-2 text-sm text-white">Pose-Erkennung lädt …</div>
	{/if}
</div>

// Lädt den MediaPipe PoseLandmarker (lokal ausgeliefert, kein CDN) als Singleton.

import type { PoseLandmarker } from '@mediapipe/tasks-vision';
import type { Point } from './geometry';

let instance: Promise<PoseLandmarker> | null = null;
let lastTs = 0;

async function create(): Promise<PoseLandmarker> {
	const { FilesetResolver, PoseLandmarker } = await import('@mediapipe/tasks-vision');
	const fileset = await FilesetResolver.forVisionTasks('/mediapipe/wasm');
	const base = {
		runningMode: 'VIDEO' as const,
		numPoses: 1,
		minPoseDetectionConfidence: 0.5,
		minPosePresenceConfidence: 0.5,
		minTrackingConfidence: 0.5
	};
	try {
		return await PoseLandmarker.createFromOptions(fileset, {
			...base,
			baseOptions: { modelAssetPath: '/models/pose_landmarker_lite.task', delegate: 'GPU' }
		});
	} catch (err) {
		console.warn('GPU-Delegate nicht verfügbar, nutze CPU', err);
		return PoseLandmarker.createFromOptions(fileset, {
			...base,
			baseOptions: { modelAssetPath: '/models/pose_landmarker_lite.task', delegate: 'CPU' }
		});
	}
}

export function loadLandmarker(): Promise<PoseLandmarker> {
	if (!instance) {
		instance = create().catch((err) => {
			instance = null;
			throw err;
		});
	}
	return instance;
}

/**
 * Erkennt die Pose in einem Videoframe. Zeitstempel müssen im VIDEO-Modus streng
 * steigen – auch beim Zurückspulen im Review – daher ein eigener monotoner Zähler.
 */
export function detect(landmarker: PoseLandmarker, video: HTMLVideoElement): Point[] | null {
	if (video.readyState < 2 || video.videoWidth === 0) return null;
	const ts = Math.max(lastTs + 1, performance.now());
	lastTs = ts;
	const result = landmarker.detectForVideo(video, ts);
	const lms = result.landmarks[0];
	if (!lms) return null;
	return lms.map((l) => ({ x: l.x, y: l.y, visibility: l.visibility }));
}

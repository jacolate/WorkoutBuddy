// Kopiert die MediaPipe-WASM-Dateien nach static/ und lädt das Pose-Modell herunter,
// damit die App komplett offline (ohne CDN) läuft.
import { copyFile, mkdir, stat, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const wasmSrc = join(root, 'node_modules/@mediapipe/tasks-vision/wasm');
const wasmDst = join(root, 'static/mediapipe/wasm');
const modelDst = join(root, 'static/models/pose_landmarker_lite.task');
const MODEL_URL =
	'https://storage.googleapis.com/mediapipe-models/pose_landmarker/pose_landmarker_lite/float16/latest/pose_landmarker_lite.task';

async function exists(p) {
	try {
		return (await stat(p)).size > 0;
	} catch {
		return false;
	}
}

await mkdir(wasmDst, { recursive: true });
for (const f of [
	'vision_wasm_internal.js',
	'vision_wasm_internal.wasm',
	'vision_wasm_nosimd_internal.js',
	'vision_wasm_nosimd_internal.wasm'
]) {
	await copyFile(join(wasmSrc, f), join(wasmDst, f));
}
console.log('MediaPipe WASM kopiert');

if (await exists(modelDst)) {
	console.log('Pose-Modell vorhanden');
} else {
	await mkdir(dirname(modelDst), { recursive: true });
	console.log('Lade Pose-Modell …');
	const res = await fetch(MODEL_URL);
	if (!res.ok) throw new Error(`Modell-Download fehlgeschlagen: ${res.status}`);
	await writeFile(modelDst, Buffer.from(await res.arrayBuffer()));
	console.log('Pose-Modell gespeichert');
}

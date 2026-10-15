// Kamerazugriff und Videoaufnahme.

export type Facing = 'user' | 'environment';

export class CameraError extends Error {
	constructor(
		message: string,
		public reason: 'insecure' | 'unsupported' | 'denied' | 'notfound' | 'other'
	) {
		super(message);
	}
}

export async function openCamera(facing: Facing): Promise<MediaStream> {
	if (!window.isSecureContext) {
		throw new CameraError(
			'Die Kamera funktioniert nur über HTTPS (oder localhost). Starte den Server mit TLS_MODE=self-signed oder hinter einem HTTPS-Proxy.',
			'insecure'
		);
	}
	if (!navigator.mediaDevices?.getUserMedia) {
		throw new CameraError('Dieser Browser unterstützt keinen Kamerazugriff.', 'unsupported');
	}
	try {
		return await navigator.mediaDevices.getUserMedia({
			audio: false,
			video: {
				facingMode: { ideal: facing },
				width: { ideal: 1280 },
				height: { ideal: 720 },
				frameRate: { ideal: 30 }
			}
		});
	} catch (err) {
		const name = (err as DOMException)?.name;
		if (name === 'NotAllowedError' || name === 'SecurityError') {
			throw new CameraError('Kamerazugriff wurde verweigert. Bitte in den Browser-Einstellungen erlauben.', 'denied');
		}
		if (name === 'NotFoundError' || name === 'OverconstrainedError') {
			throw new CameraError('Keine passende Kamera gefunden.', 'notfound');
		}
		throw new CameraError(`Kamera konnte nicht gestartet werden: ${(err as Error)?.message ?? err}`, 'other');
	}
}

export function stopStream(stream: MediaStream | null | undefined) {
	stream?.getTracks().forEach((t) => t.stop());
}

/** Bestes unterstütztes Aufnahmeformat (Safari: mp4, Chrome/Firefox: webm). */
export function pickMimeType(): string {
	const candidates = [
		'video/mp4;codecs=avc1',
		'video/mp4',
		'video/webm;codecs=vp9',
		'video/webm;codecs=vp8',
		'video/webm'
	];
	if (typeof MediaRecorder === 'undefined') return '';
	return candidates.find((c) => MediaRecorder.isTypeSupported(c)) ?? '';
}

export function canRecord(): boolean {
	return typeof MediaRecorder !== 'undefined' && pickMimeType() !== '';
}

export interface Clip {
	blob: Blob;
	mime: string;
	durationSec: number;
}

/** Nimmt einen MediaStream auf, bis stop() aufgerufen wird. */
export class ClipRecorder {
	#rec: MediaRecorder;
	#chunks: Blob[] = [];
	#start = 0;
	#done: Promise<Clip>;
	readonly mime: string;

	constructor(stream: MediaStream) {
		this.mime = pickMimeType();
		this.#rec = new MediaRecorder(stream, {
			mimeType: this.mime || undefined,
			videoBitsPerSecond: 2_500_000
		});
		this.#done = new Promise((resolve, reject) => {
			this.#rec.ondataavailable = (e) => {
				if (e.data.size > 0) this.#chunks.push(e.data);
			};
			this.#rec.onstop = () => {
				const mime = (this.#rec.mimeType || this.mime || 'video/webm').split(';')[0];
				resolve({
					blob: new Blob(this.#chunks, { type: mime }),
					mime,
					durationSec: (performance.now() - this.#start) / 1000
				});
			};
			this.#rec.onerror = (e) => reject((e as unknown as { error?: Error }).error ?? new Error('Aufnahmefehler'));
		});
	}

	start() {
		this.#start = performance.now();
		// Zeitscheiben, damit bei einem Absturz nicht alles verloren ist.
		this.#rec.start(1000);
	}

	get recording() {
		return this.#rec.state === 'recording';
	}

	stop(): Promise<Clip> {
		if (this.#rec.state !== 'inactive') this.#rec.stop();
		return this.#done;
	}
}

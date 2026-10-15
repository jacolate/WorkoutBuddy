// Hintergrund-Upload von Aufnahmen mit Wiederholungsversuchen.
import { api } from '#lib/api.ts';
import type { Clip } from './camera';

export interface UploadMeta {
	sessionId: number | null;
	exerciseId: number | null;
	label: string;
}

export interface UploadJob {
	id: number;
	label: string;
	status: 'uploading' | 'done' | 'error';
	error?: string;
	recordingId?: number;
	clip: Clip;
	meta: UploadMeta;
}

let nextId = 1;

class Uploads {
	jobs = $state<UploadJob[]>([]);

	get pending() {
		return this.jobs.filter((j) => j.status === 'uploading').length;
	}

	get failed() {
		return this.jobs.filter((j) => j.status === 'error');
	}

	enqueue(clip: Clip, meta: UploadMeta) {
		const job: UploadJob = { id: nextId++, label: meta.label, status: 'uploading', clip, meta };
		this.jobs.push(job);
		void this.#run(job.id);
	}

	retry(id: number) {
		const job = this.jobs.find((j) => j.id === id);
		if (!job) return;
		job.status = 'uploading';
		job.error = undefined;
		void this.#run(id);
	}

	discard(id: number) {
		this.jobs = this.jobs.filter((j) => j.id !== id);
	}

	/** Fehlgeschlagene Aufnahme lokal speichern, damit sie nicht verloren geht. */
	download(id: number) {
		const job = this.jobs.find((j) => j.id === id);
		if (!job) return;
		const url = URL.createObjectURL(job.clip.blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${job.label.replace(/[^\wäöüÄÖÜß-]+/g, '_')}.${job.clip.mime.includes('mp4') ? 'mp4' : 'webm'}`;
		a.click();
		setTimeout(() => URL.revokeObjectURL(url), 10_000);
	}

	async #run(id: number) {
		const job = () => this.jobs.find((j) => j.id === id);
		const j0 = job();
		if (!j0) return;
		const { clip, meta } = j0;
		let lastErr: unknown;
		for (let attempt = 0; attempt < 3; attempt++) {
			try {
				const form = new FormData();
				if (meta.sessionId) form.append('sessionId', String(meta.sessionId));
				if (meta.exerciseId) form.append('exerciseId', String(meta.exerciseId));
				form.append('durationSec', clip.durationSec.toFixed(1));
				form.append('label', meta.label);
				form.append('file', clip.blob, 'clip');
				const rec = await api.uploadRecording(form);
				const j = job();
				if (j) {
					j.status = 'done';
					j.recordingId = rec.id;
				}
				return;
			} catch (err) {
				lastErr = err;
				await new Promise((r) => setTimeout(r, 1500 * (attempt + 1)));
			}
		}
		const j = job();
		if (j) {
			j.status = 'error';
			j.error = (lastErr as Error)?.message ?? 'Upload fehlgeschlagen';
		}
	}
}

export const uploads = new Uploads();

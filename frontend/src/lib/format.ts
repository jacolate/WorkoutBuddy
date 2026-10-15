import type { LogEntry, Side, Weight } from '#lib/types.ts';

export function formatWeight(w: Weight): string {
	if (w === 'bw') return 'Körpergewicht';
	if (w === null || w === undefined) return '';
	return `${formatNumber(w)} kg`;
}

export function formatWeightShort(w: Weight): string {
	if (w === 'bw') return 'KG';
	if (w === null || w === undefined) return '–';
	return `${formatNumber(w)}`;
}

export function formatNumber(n: number): string {
	return Number.isInteger(n) ? String(n) : n.toFixed(1).replace('.', ',');
}

/** 75 → „1:15“, 3600 → „1:00:00“ */
export function formatClock(totalSec: number): string {
	const s = Math.max(0, Math.round(totalSec));
	const h = Math.floor(s / 3600);
	const m = Math.floor((s % 3600) / 60);
	const sec = s % 60;
	const pad = (n: number) => String(n).padStart(2, '0');
	return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${m}:${pad(sec)}`;
}

/** 75 → „1 min 15 s“, 45 → „45 s“ */
export function formatDuration(totalSec: number): string {
	const s = Math.max(0, Math.round(totalSec));
	if (s < 60) return `${s} s`;
	const h = Math.floor(s / 3600);
	const m = Math.floor((s % 3600) / 60);
	const sec = s % 60;
	if (h > 0) return `${h} h ${m} min`;
	// Ab 10 Minuten sind Sekunden nur Scheingenauigkeit.
	if (s >= 600) return `${Math.round(s / 60)} min`;
	return sec ? `${m} min ${sec} s` : `${m} min`;
}

export function formatSide(side?: Side): string {
	if (side === 'L') return 'Links';
	if (side === 'R') return 'Rechts';
	return '';
}

const dateFmt = new Intl.DateTimeFormat('de-DE', { weekday: 'short', day: '2-digit', month: '2-digit', year: 'numeric' });
const timeFmt = new Intl.DateTimeFormat('de-DE', { hour: '2-digit', minute: '2-digit' });

export function formatDate(iso: string): string {
	return dateFmt.format(new Date(iso));
}

export function formatDateTime(iso: string): string {
	const d = new Date(iso);
	return `${dateFmt.format(d)}, ${timeFmt.format(d)}`;
}

export function relativeDay(iso: string): string {
	const d = new Date(iso);
	const today = new Date();
	const startOf = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
	const diff = Math.round((startOf(today) - startOf(d)) / 86_400_000);
	if (diff === 0) return 'Heute';
	if (diff === 1) return 'Gestern';
	if (diff < 7) return `vor ${diff} Tagen`;
	return formatDate(iso);
}

export function formatBytes(n: number): string {
	if (n < 1024) return `${n} B`;
	if (n < 1024 ** 2) return `${(n / 1024).toFixed(0)} KB`;
	if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
	return `${(n / 1024 ** 3).toFixed(2)} GB`;
}

/** Zusammenfassung eines Session-Logs. */
export function summarizeLog(log: LogEntry[]): { reps: number; volumeKg: number; workSec: number; exercises: number } {
	let reps = 0;
	let volumeKg = 0;
	let workSec = 0;
	const ex = new Set<number>();
	for (const e of log) {
		if (e.skipped) continue;
		workSec += e.durationSec;
		if (e.exerciseId !== null) ex.add(e.exerciseId);
		if (e.reps) {
			reps += e.reps;
			if (typeof e.weight === 'number') volumeKg += e.reps * e.weight;
		}
	}
	return { reps, volumeKg, workSec, exercises: ex.size };
}

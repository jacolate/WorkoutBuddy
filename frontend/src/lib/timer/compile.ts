// Übersetzt eine Workout-Struktur in eine flache Liste von Schritten, die der Player abspielt.
// Reine Funktion ohne Seiteneffekte → gut testbar.

import type { Block, BlockItem, Exercise, Side, Weight, WorkoutStructure } from '#lib/types.ts';

export type StepKind = 'prepare' | 'work' | 'rest';

export interface AmrapItem {
	exerciseId: number;
	name: string;
	reps: number | null;
	weight: Weight;
	unilateral: boolean;
}

export interface Step {
	index: number;
	kind: StepKind;
	/** 'timer' = läuft ab, 'manual' = endet per „Fertig“ */
	advance: 'timer' | 'manual';
	durationSec: number | null;
	exerciseId: number | null;
	exerciseName: string;
	targetReps: number | null;
	targetDistanceM: number | null;
	weight: Weight;
	side?: Side;
	amrapItems?: AmrapItem[];
	blockIndex: number;
	blockLabel: string;
	/** z. B. „Satz 2/5“, „Runde 3/8“, „Minute 4/12“ */
	progress: string;
	/** Kurzer Text für Pausen, z. B. „Seitenwechsel“ */
	note?: string;
	record: boolean;
}

export type ExerciseLookup = Map<number, Exercise>;

export const SIDE_SWITCH_SEC = 5;

export const BLOCK_TYPE_LABELS: Record<Block['type'], string> = {
	sets: 'Sätze',
	circuit: 'Zirkel',
	interval: 'Intervall',
	emom: 'EMOM',
	amrap: 'AMRAP',
	rest: 'Pause'
};

export function blockLabel(block: Block): string {
	return block.title?.trim() || BLOCK_TYPE_LABELS[block.type];
}

type Draft = Omit<Step, 'index'>;

export function compile(structure: WorkoutStructure, exercises: ExerciseLookup): Step[] {
	const out: Draft[] = [];
	const blocks = structure.blocks;

	if (structure.prepareSec > 0) {
		out.push(timerStep('prepare', structure.prepareSec, -1, 'Vorbereitung', '', { note: 'Gleich geht’s los' }));
	}

	let lastEmittedBlock: Block | null = null;
	blocks.forEach((block, bi) => {
		const steps = compileBlock(block, bi, exercises);
		if (steps.length === 0) return;
		if (
			lastEmittedBlock &&
			structure.blockRestSec > 0 &&
			lastEmittedBlock.type !== 'rest' &&
			block.type !== 'rest'
		) {
			out.push(timerStep('rest', structure.blockRestSec, bi, 'Pause', '', { note: 'Blockwechsel' }));
		}
		out.push(...steps);
		lastEmittedBlock = block;
	});

	// Pausen am Ende sind sinnlos.
	while (out.length > 0 && out[out.length - 1].kind === 'rest') out.pop();
	// Nur Vorbereitung ohne Inhalt → leer.
	if (out.length === 1 && out[0].kind === 'prepare') out.pop();

	return out.map((s, index) => ({ ...s, index }));
}

function timerStep(kind: StepKind, durationSec: number, blockIndex: number, label: string, progress: string, extra: Partial<Draft> = {}): Draft {
	return {
		kind,
		advance: 'timer',
		durationSec,
		exerciseId: null,
		exerciseName: '',
		targetReps: null,
		targetDistanceM: null,
		weight: null,
		blockIndex,
		blockLabel: label,
		progress,
		record: false,
		...extra
	};
}

function lookup(exercises: ExerciseLookup, id: number): { name: string; unilateral: boolean } {
	const e = exercises.get(id);
	return e ? { name: e.name, unilateral: e.unilateral } : { name: 'Unbekannte Übung', unilateral: false };
}

/** Arbeitsschritt für ein Item; Reps/Strecke → manuell, sonst Zeit. */
function workStep(item: BlockItem, exercises: ExerciseLookup, bi: number, label: string, progress: string, record: boolean, side?: Side): Draft {
	const ex = lookup(exercises, item.exerciseId);
	const timed = !item.reps && !item.distanceM && !!item.durationSec;
	return {
		kind: 'work',
		advance: timed ? 'timer' : 'manual',
		durationSec: timed ? item.durationSec! : null,
		exerciseId: item.exerciseId,
		exerciseName: ex.name,
		targetReps: item.reps ?? null,
		targetDistanceM: item.reps ? null : (item.distanceM ?? null),
		weight: item.weight,
		side,
		blockIndex: bi,
		blockLabel: label,
		progress,
		record
	};
}

/** Expandiert ein Item in L/R-Schritte, falls die Übung einseitig ist. */
function sidedWork(item: BlockItem, exercises: ExerciseLookup, bi: number, label: string, progress: string, record: boolean): Draft[] {
	const { unilateral } = lookup(exercises, item.exerciseId);
	if (!unilateral) return [workStep(item, exercises, bi, label, progress, record)];
	const left = workStep(item, exercises, bi, label, progress, record, 'L');
	const right = workStep(item, exercises, bi, label, progress, record, 'R');
	if (left.advance === 'timer') {
		return [left, timerStep('rest', SIDE_SWITCH_SEC, bi, label, progress, { note: 'Seitenwechsel' }), right];
	}
	return [left, right];
}

function compileBlock(block: Block, bi: number, exercises: ExerciseLookup): Draft[] {
	const label = blockLabel(block);
	const record = !!block.record;
	const out: Draft[] = [];

	switch (block.type) {
		case 'sets': {
			for (let s = 1; s <= block.sets; s++) {
				out.push(...sidedWork(block.item, exercises, bi, label, `Satz ${s}/${block.sets}`, record));
				if (s < block.sets && block.restSec > 0) {
					out.push(timerStep('rest', block.restSec, bi, label, `Satz ${s}/${block.sets}`));
				}
			}
			break;
		}
		case 'circuit': {
			if (block.items.length === 0) break;
			for (let r = 1; r <= block.rounds; r++) {
				const progress = `Runde ${r}/${block.rounds}`;
				block.items.forEach((item, ii) => {
					out.push(...sidedWork(item, exercises, bi, label, progress, record));
					const lastItem = ii === block.items.length - 1;
					if (!lastItem && block.restBetweenItemsSec > 0) {
						out.push(timerStep('rest', block.restBetweenItemsSec, bi, label, progress));
					}
				});
				if (r < block.rounds && block.restBetweenRoundsSec > 0) {
					out.push(timerStep('rest', block.restBetweenRoundsSec, bi, label, progress, { note: 'Rundenpause' }));
				}
			}
			break;
		}
		case 'interval': {
			if (block.items.length === 0 || block.workSec <= 0) break;
			const total = block.rounds * block.items.length;
			let n = 0;
			for (let r = 1; r <= block.rounds; r++) {
				block.items.forEach((item) => {
					n++;
					const ex = lookup(exercises, item.exerciseId);
					const side: Side | undefined = ex.unilateral ? (r % 2 === 1 ? 'L' : 'R') : undefined;
					const progress = `Runde ${r}/${block.rounds}`;
					out.push({
						...workStep({ ...item, durationSec: block.workSec }, exercises, bi, label, progress, record, side),
						advance: 'timer',
						durationSec: block.workSec,
						targetDistanceM: null
					});
					if (n < total && block.restSec > 0) {
						out.push(timerStep('rest', block.restSec, bi, label, progress));
					}
				});
			}
			break;
		}
		case 'emom': {
			if (block.items.length === 0) break;
			const appearances = new Map<number, number>();
			for (let m = 1; m <= block.minutes; m++) {
				const ii = (m - 1) % block.items.length;
				const item = block.items[ii];
				const ex = lookup(exercises, item.exerciseId);
				const k = appearances.get(ii) ?? 0;
				appearances.set(ii, k + 1);
				const side: Side | undefined = ex.unilateral ? (k % 2 === 0 ? 'L' : 'R') : undefined;
				out.push({
					...workStep(item, exercises, bi, label, `Minute ${m}/${block.minutes}`, record, side),
					advance: 'timer',
					durationSec: 60
				});
			}
			break;
		}
		case 'amrap': {
			if (block.items.length === 0 || block.durationSec <= 0) break;
			const items: AmrapItem[] = block.items.map((it) => {
				const ex = lookup(exercises, it.exerciseId);
				return { exerciseId: it.exerciseId, name: ex.name, reps: it.reps ?? null, weight: it.weight, unilateral: ex.unilateral };
			});
			out.push({
				kind: 'work',
				advance: 'timer',
				durationSec: block.durationSec,
				exerciseId: null,
				exerciseName: 'AMRAP',
				targetReps: null,
				targetDistanceM: null,
				weight: null,
				amrapItems: items,
				blockIndex: bi,
				blockLabel: label,
				progress: `${Math.round(block.durationSec / 60)} min`,
				record
			});
			break;
		}
		case 'rest': {
			if (block.durationSec > 0) out.push(timerStep('rest', block.durationSec, bi, label, ''));
			break;
		}
	}
	return out;
}

/** Geschätzte Gesamtdauer in Sekunden (manuelle Schritte: Reps × secondsPerRep, ~1 s pro Meter). */
export function estimateDuration(steps: Step[], secondsPerRep = 3): number {
	let total = 0;
	for (const s of steps) {
		if (s.advance === 'timer') total += s.durationSec ?? 0;
		else if (s.targetDistanceM) total += s.targetDistanceM;
		else total += (s.targetReps ?? 10) * secondsPerRep;
	}
	return total;
}

/** Gesamtzahl geplanter Wiederholungen und Volumen (Reps × kg). AMRAP zählt nicht mit. */
export function plannedVolume(steps: Step[]): { reps: number; kg: number } {
	let reps = 0;
	let kg = 0;
	for (const s of steps) {
		if (s.kind !== 'work' || !s.targetReps) continue;
		reps += s.targetReps;
		if (typeof s.weight === 'number') kg += s.targetReps * s.weight;
	}
	return { reps, kg };
}

import { describe, expect, it } from 'vitest';
import { compile, estimateDuration, plannedVolume, SIDE_SWITCH_SEC, type ExerciseLookup } from './compile';
import type { Block, Exercise, WorkoutStructure } from '#lib/types.ts';

function ex(id: number, name: string, unilateral = false): Exercise {
	return { id, name, description: '', category: 'x', unilateral, defaultWeightKg: 16, poseProfile: null, cues: '', cameraHint: '' };
}

const exercises: ExerciseLookup = new Map([
	[1, ex(1, 'Swing')],
	[2, ex(2, 'Press', true)],
	[3, ex(3, 'Goblet Squat')],
	[4, ex(4, 'Plank')]
]);

function structure(blocks: Block[], prepareSec = 0, blockRestSec = 0): WorkoutStructure {
	return { version: 1, prepareSec, blockRestSec, blocks };
}

const kinds = (s: { kind: string }[]) => s.map((x) => x.kind).join(',');

describe('compile', () => {
	it('sets mit Reps → manuelle Schritte mit Pausen dazwischen, nicht am Ende', () => {
		const steps = compile(
			structure([{ id: 'a', type: 'sets', sets: 3, restSec: 60, item: { exerciseId: 1, reps: 15, weight: 16 } }]),
			exercises
		);
		expect(kinds(steps)).toBe('work,rest,work,rest,work');
		expect(steps[0]).toMatchObject({ advance: 'manual', targetReps: 15, weight: 16, exerciseName: 'Swing', progress: 'Satz 1/3' });
		expect(steps[1]).toMatchObject({ advance: 'timer', durationSec: 60 });
		expect(steps.map((s) => s.index)).toEqual([0, 1, 2, 3, 4]);
	});

	it('unilaterale Übung mit Reps → L und R direkt hintereinander', () => {
		const steps = compile(
			structure([{ id: 'a', type: 'sets', sets: 2, restSec: 30, item: { exerciseId: 2, reps: 5, weight: 12 } }]),
			exercises
		);
		expect(steps.map((s) => s.side ?? '-').join('')).toBe('LR-LR');
		expect(kinds(steps)).toBe('work,work,rest,work,work');
	});

	it('unilaterale Zeit-Übung → Seitenwechsel-Pause', () => {
		const steps = compile(
			structure([{ id: 'a', type: 'sets', sets: 1, restSec: 0, item: { exerciseId: 2, durationSec: 30, weight: 12 } }]),
			exercises
		);
		expect(kinds(steps)).toBe('work,rest,work');
		expect(steps[1]).toMatchObject({ durationSec: SIDE_SWITCH_SEC, note: 'Seitenwechsel' });
		expect(steps[0]).toMatchObject({ advance: 'timer', durationSec: 30, side: 'L' });
	});

	it('Tabata: 8 Runden 20/10 ohne Pause am Ende', () => {
		const steps = compile(
			structure([{ id: 't', type: 'interval', rounds: 8, workSec: 20, restSec: 10, items: [{ exerciseId: 1, weight: 16 }] }]),
			exercises
		);
		expect(steps.filter((s) => s.kind === 'work')).toHaveLength(8);
		expect(steps.filter((s) => s.kind === 'rest')).toHaveLength(7);
		expect(estimateDuration(steps)).toBe(8 * 20 + 7 * 10);
		expect(steps.every((s) => s.advance === 'timer')).toBe(true);
	});

	it('Intervall mit unilateraler Übung wechselt die Seite pro Runde', () => {
		const steps = compile(
			structure([{ id: 't', type: 'interval', rounds: 4, workSec: 30, restSec: 15, items: [{ exerciseId: 2, weight: 12 }] }]),
			exercises
		);
		expect(steps.filter((s) => s.kind === 'work').map((s) => s.side).join('')).toBe('LRLR');
	});

	it('EMOM rotiert Übungen pro Minute, unilaterale wechseln pro Auftreten', () => {
		const steps = compile(
			structure([
				{
					id: 'e',
					type: 'emom',
					minutes: 6,
					items: [
						{ exerciseId: 2, reps: 5, weight: 12 },
						{ exerciseId: 3, reps: 8, weight: 16 }
					]
				}
			]),
			exercises
		);
		expect(steps).toHaveLength(6);
		expect(steps.map((s) => s.exerciseId)).toEqual([2, 3, 2, 3, 2, 3]);
		expect(steps.filter((s) => s.exerciseId === 2).map((s) => s.side)).toEqual(['L', 'R', 'L']);
		expect(steps.every((s) => s.durationSec === 60 && s.advance === 'timer')).toBe(true);
		expect(steps[5].progress).toBe('Minute 6/6');
		expect(steps[0].targetReps).toBe(5);
	});

	it('AMRAP ist ein Schritt mit Item-Liste', () => {
		const steps = compile(
			structure([
				{
					id: 'a',
					type: 'amrap',
					durationSec: 600,
					items: [
						{ exerciseId: 1, reps: 15, weight: 16 },
						{ exerciseId: 2, reps: 5, weight: 12 }
					]
				}
			]),
			exercises
		);
		expect(steps).toHaveLength(1);
		expect(steps[0].amrapItems).toHaveLength(2);
		expect(steps[0].amrapItems![1]).toMatchObject({ name: 'Press', unilateral: true });
	});

	it('Zirkel: Pausen zwischen Items und Runden', () => {
		const steps = compile(
			structure([
				{
					id: 'c',
					type: 'circuit',
					rounds: 2,
					restBetweenItemsSec: 10,
					restBetweenRoundsSec: 60,
					items: [
						{ exerciseId: 1, reps: 10, weight: 12 },
						{ exerciseId: 3, reps: 5, weight: 12 }
					]
				}
			]),
			exercises
		);
		expect(kinds(steps)).toBe('work,rest,work,rest,work,rest,work');
		expect(steps[3]).toMatchObject({ durationSec: 60, note: 'Rundenpause' });
	});

	it('Vorbereitung, Blockpausen und Rest-Blöcke werden korrekt kombiniert', () => {
		const steps = compile(
			structure(
				[
					{ id: 'a', type: 'sets', sets: 1, restSec: 0, item: { exerciseId: 1, reps: 10, weight: 16 } },
					{ id: 'b', type: 'sets', sets: 1, restSec: 0, item: { exerciseId: 3, reps: 10, weight: 16 } },
					{ id: 'r', type: 'rest', durationSec: 120 },
					{ id: 'c', type: 'sets', sets: 1, restSec: 0, item: { exerciseId: 4, durationSec: 40, weight: 'bw' } }
				],
				10,
				60
			),
			exercises
		);
		// prepare, a, Blockpause, b, Rest-Block (keine zusätzliche Blockpause), c
		expect(kinds(steps)).toBe('prepare,work,rest,work,rest,work');
		expect(steps[2]).toMatchObject({ durationSec: 60, note: 'Blockwechsel' });
		expect(steps[4]).toMatchObject({ durationSec: 120 });
	});

	it('leere Blöcke erzeugen nichts, auch keine Blockpausen; Vorbereitung allein entfällt', () => {
		expect(compile(structure([{ id: 'x', type: 'circuit', rounds: 3, items: [], restBetweenItemsSec: 0, restBetweenRoundsSec: 30 }], 10, 60), exercises)).toEqual([]);
		const steps = compile(
			structure(
				[
					{ id: 'x', type: 'amrap', durationSec: 0, items: [] },
					{ id: 'a', type: 'sets', sets: 1, restSec: 0, item: { exerciseId: 1, reps: 10, weight: 16 } },
					{ id: 'r', type: 'rest', durationSec: 60 }
				],
				0,
				60
			),
			exercises
		);
		expect(kinds(steps)).toBe('work');
	});

	it('Volumen und Dauerschätzung', () => {
		const steps = compile(
			structure([{ id: 'a', type: 'sets', sets: 2, restSec: 30, item: { exerciseId: 2, reps: 5, weight: 12 } }]),
			exercises
		);
		// 2 Sätze × 2 Seiten × 5 Reps
		expect(plannedVolume(steps)).toEqual({ reps: 20, kg: 240 });
		expect(estimateDuration(steps, 3)).toBe(20 * 3 + 30);
	});

	it('unbekannte Übung bricht nicht', () => {
		const steps = compile(structure([{ id: 'a', type: 'sets', sets: 1, restSec: 0, item: { exerciseId: 99, reps: 5, weight: null } }]), exercises);
		expect(steps[0].exerciseName).toBe('Unbekannte Übung');
	});
});

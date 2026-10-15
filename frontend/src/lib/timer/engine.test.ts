import { describe, expect, it } from 'vitest';
import { PlayerEngine } from './engine.svelte';
import type { Step } from './compile';

function step(i: number, kind: Step['kind'], advance: Step['advance'], durationSec: number | null): Step {
	return {
		index: i,
		kind,
		advance,
		durationSec,
		exerciseId: kind === 'work' ? 1 : null,
		exerciseName: kind === 'work' ? 'Swing' : '',
		targetReps: advance === 'manual' ? 10 : null,
		targetDistanceM: null,
		weight: 16,
		blockIndex: 0,
		blockLabel: 'Test',
		progress: '',
		record: false
	};
}

function setup(steps: Step[]) {
	let t = 0;
	const log: string[] = [];
	const ends: { idx: number; elapsed: number; skipped: boolean }[] = [];
	const engine = new PlayerEngine(
		steps,
		{
			onStepStart: (s, { catchUp }) => log.push(`start:${s.index}${catchUp ? ':catchup' : ''}`),
			onStepEnd: (s, info) => ends.push({ idx: s.index, elapsed: info.elapsedSec, skipped: info.skipped }),
			onSecond: (sec) => log.push(`sec:${sec}`),
			onFinish: () => log.push('finish')
		},
		() => t
	);
	return {
		engine,
		log,
		ends,
		advance(ms: number) {
			t += ms;
			engine.tick();
		}
	};
}

describe('PlayerEngine', () => {
	it('läuft Timer-Schritte automatisch ab und meldet Sekunden', () => {
		const { engine, log, advance } = setup([step(0, 'prepare', 'timer', 3), step(1, 'work', 'timer', 2)]);
		engine.start();
		expect(engine.remainingSec).toBe(3);
		advance(1000);
		advance(1000);
		advance(1000);
		expect(engine.index).toBe(1);
		advance(1000);
		advance(1000);
		expect(engine.status).toBe('finished');
		expect(log).toEqual(['start:0', 'sec:3', 'sec:2', 'sec:1', 'start:1', 'sec:2', 'sec:1', 'finish']);
	});

	it('manuelle Schritte warten auf complete()', () => {
		const { engine, ends, advance } = setup([step(0, 'work', 'manual', null), step(1, 'rest', 'timer', 30)]);
		engine.start();
		advance(60_000);
		expect(engine.index).toBe(0);
		expect(engine.stepElapsedMs).toBe(60_000);
		engine.complete();
		expect(engine.index).toBe(1);
		expect(ends[0]).toEqual({ idx: 0, elapsed: 60, skipped: false });
		expect(engine.remainingSec).toBe(30);
	});

	it('Pause friert die Zeit ein', () => {
		const { engine, advance } = setup([step(0, 'work', 'timer', 10)]);
		engine.start();
		advance(4000);
		engine.pause();
		advance(60_000);
		expect(engine.remainingSec).toBe(6);
		engine.resume();
		advance(1000);
		expect(engine.remainingSec).toBe(5);
		expect(engine.totalElapsedMs).toBe(5000);
	});

	it('holt nach langer Hintergrundzeit auf und überträgt die Restzeit', () => {
		const { engine, log, ends, advance } = setup([
			step(0, 'work', 'timer', 20),
			step(1, 'rest', 'timer', 10),
			step(2, 'work', 'timer', 20),
			step(3, 'work', 'manual', null)
		]);
		engine.start();
		advance(35_000); // 20 + 10 vorbei, 5 s in Schritt 2
		expect(engine.index).toBe(2);
		expect(engine.remainingSec).toBe(15);
		expect(ends.map((e) => e.idx)).toEqual([0, 1]);
		expect(log).toContain('start:1:catchup');
		expect(log).toContain('start:2');
		advance(100_000); // bleibt beim manuellen Schritt stehen
		expect(engine.index).toBe(3);
		expect(engine.status).toBe('running');
	});

	it('skip markiert übersprungen, back startet Schritt neu bzw. geht zurück', () => {
		const { engine, ends, advance } = setup([step(0, 'work', 'timer', 30), step(1, 'rest', 'timer', 30), step(2, 'work', 'timer', 30)]);
		engine.start();
		advance(1000);
		engine.skip();
		expect(ends[0].skipped).toBe(true);
		expect(engine.index).toBe(1);
		advance(1000);
		engine.back(); // < 3 s → vorheriger Schritt
		expect(engine.index).toBe(0);
		expect(engine.remainingSec).toBe(30);
		advance(10_000);
		engine.back(); // > 3 s → Neustart
		expect(engine.index).toBe(0);
		advance(0);
		expect(engine.remainingSec).toBe(30);
	});

	it('complete im Pausenzustand funktioniert und bleibt pausiert', () => {
		const { engine, advance } = setup([step(0, 'work', 'manual', null), step(1, 'work', 'timer', 10)]);
		engine.start();
		advance(5000);
		engine.pause();
		advance(5000);
		engine.complete();
		expect(engine.index).toBe(1);
		expect(engine.status).toBe('paused');
		engine.resume();
		advance(2000);
		expect(engine.remainingSec).toBe(8);
	});

	it('leeres Workout ist sofort fertig', () => {
		const { engine, log } = setup([]);
		engine.start();
		expect(engine.status).toBe('finished');
		expect(log).toEqual(['finish']);
	});
});

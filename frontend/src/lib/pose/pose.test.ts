import { describe, expect, it } from 'vitest';
import { angle, angleFromVertical, computeMetrics, LM, type Point } from './geometry';
import { RepCounter } from './repCounter';
import { PoseAnalyzer } from './analyzer';

const deg = (d: number) => (d * Math.PI) / 180;

/**
 * Synthetisches Seitenansicht-Skelett (Blick nach rechts, y nach unten).
 * kneeBend 0 = Stand, 1 = tiefe Hocke. hinge = zusätzliche Hüftbeugung durch Rumpfneigung.
 */
function skeleton({ kneeBend = 0, lean = 10, armUp = 0 }: { kneeBend?: number; lean?: number; armUp?: number }): Point[] {
	const lms: Point[] = Array.from({ length: 33 }, () => ({ x: 0.5, y: 0.5, visibility: 0.95 }));
	const L = 0.2;
	const ankle = { x: 0.5, y: 0.9 };
	const shinA = deg(35 * kneeBend);
	const knee = { x: ankle.x + Math.sin(shinA) * L, y: ankle.y - Math.cos(shinA) * L };
	// Oberschenkel: im Stand senkrecht, in der Hocke waagerecht nach hinten
	const thighA = deg(-95 * kneeBend);
	const hip = { x: knee.x + Math.sin(thighA) * L, y: knee.y - Math.cos(thighA) * L };
	const leanA = deg(lean);
	const shoulder = { x: hip.x + Math.sin(leanA) * 0.3, y: hip.y - Math.cos(leanA) * 0.3 };
	// Arm: armUp 0 = Rack (Ellbogen ~45°, Handgelenk auf Schulterhöhe), 1 = Lockout senkrecht über Kopf
	const phi = deg(20 + 160 * armUp); // Oberarm, 0 = nach unten, 180 = nach oben
	const elbowAngle = deg(45 + 135 * armUp);
	const elbow = { x: shoulder.x + Math.sin(phi) * 0.13, y: shoulder.y + Math.cos(phi) * 0.13 };
	const psi = phi + Math.PI - elbowAngle;
	const wrist = { x: elbow.x + Math.sin(psi) * 0.13, y: elbow.y + Math.cos(psi) * 0.13 };
	const set = (l: number, r: number, p: { x: number; y: number }) => {
		lms[l] = { ...p, visibility: 0.95 };
		lms[r] = { ...p, visibility: 0.9 };
	};
	set(LM.leftAnkle, LM.rightAnkle, ankle);
	set(LM.leftKnee, LM.rightKnee, knee);
	set(LM.leftHip, LM.rightHip, hip);
	set(LM.leftShoulder, LM.rightShoulder, shoulder);
	set(LM.leftElbow, LM.rightElbow, elbow);
	set(LM.leftWrist, LM.rightWrist, wrist);
	return lms;
}

describe('geometry', () => {
	it('angle', () => {
		expect(angle({ x: 1, y: 0 }, { x: 0, y: 0 }, { x: 0, y: 1 })).toBeCloseTo(90);
		expect(angle({ x: -1, y: 0 }, { x: 0, y: 0 }, { x: 1, y: 0 })).toBeCloseTo(180);
		expect(angleFromVertical({ x: 0, y: 1 }, { x: 0, y: 0 })).toBeCloseTo(0);
		expect(angleFromVertical({ x: 0, y: 0 }, { x: 1, y: 1 })).toBeCloseTo(45);
	});

	it('Stand vs. Hocke', () => {
		const stand = computeMetrics(skeleton({ kneeBend: 0 }), 1)!;
		const deep = computeMetrics(skeleton({ kneeBend: 1, lean: 35 }), 1)!;
		expect(stand.kneeAngle).toBeGreaterThan(170);
		expect(stand.hipBelowKnee).toBeLessThan(-0.9);
		expect(deep.kneeAngle).toBeLessThan(90);
		expect(deep.hipBelowKnee).toBeGreaterThan(-0.05);
		expect(deep.legsVisible).toBe(true);
		expect(deep.legSide).toBe('left');
	});

	it('zu wenige Landmarks → null', () => {
		expect(computeMetrics([], 1)).toBeNull();
	});
});

describe('RepCounter', () => {
	function wave(counter: RepCounter<null>, reps: number, lowest: number, periodMs = 2000, fps = 30) {
		const results = [];
		const n = Math.round((reps * periodMs * fps) / 1000);
		for (let i = 0; i <= n + fps; i++) {
			const t = (i * 1000) / fps;
			const phase = Math.min(t / periodMs, reps) * 2 * Math.PI;
			// 170 im Stand, „lowest“ am tiefsten Punkt
			const v = 170 - ((170 - lowest) * (1 - Math.cos(phase))) / 2;
			results.push(counter.push({ t, signal: v, data: null }));
		}
		return results;
	}

	it('zählt saubere Wiederholungen', () => {
		const c = new RepCounter<null>({ restIsHigh: true, activeThreshold: 125, restThreshold: 155 });
		const res = wave(c, 5, 80);
		expect(c.count).toBe(5);
		expect(res.filter((r) => r.completed).length).toBe(5);
	});

	it('ignoriert Teilbewegungen, die die Schwelle nicht erreichen', () => {
		const c = new RepCounter<null>({ restIsHigh: true, activeThreshold: 125, restThreshold: 155 });
		wave(c, 4, 135);
		expect(c.count).toBe(0);
	});

	it('Zittern um eine Schwelle zählt nicht doppelt', () => {
		const c = new RepCounter<null>({ restIsHigh: true, activeThreshold: 125, restThreshold: 155 });
		let t = 0;
		for (const v of [170, 124, 126, 124, 126, 100, 154, 156, 154, 156, 170]) {
			c.push({ t: (t += 100), signal: v, data: null });
		}
		expect(c.count).toBe(1);
	});

	it('zu schnelle Ausschläge (Rauschen) zählen nicht', () => {
		const c = new RepCounter<null>({ restIsHigh: true, activeThreshold: 125, restThreshold: 155, minRepMs: 350 });
		c.push({ t: 0, signal: 170, data: null });
		c.push({ t: 33, signal: 100, data: null });
		c.push({ t: 66, signal: 170, data: null });
		expect(c.count).toBe(0);
	});

	it('invertierte Richtung (Ruhe = niedrig, z. B. Press)', () => {
		const c = new RepCounter<null>({ restIsHigh: false, activeThreshold: 140, restThreshold: 100 });
		let t = 0;
		for (let r = 0; r < 3; r++) for (const v of [60, 90, 120, 150, 170, 150, 120, 90, 60]) c.push({ t: (t += 100), signal: v, data: null });
		expect(c.count).toBe(3);
	});
});

describe('PoseAnalyzer', () => {
	function run(analyzer: PoseAnalyzer, frames: Point[][], fps = 30) {
		const results = [];
		frames.forEach((f, i) => {
			const out = analyzer.process(f, (i * 1000) / fps, 1);
			if (out.result) results.push(out.result);
		});
		const last = analyzer.flush();
		if (last) results.push(last);
		return analyzer.results;
	}

	function squatFrames(reps: number, maxBend: number, lean = 30) {
		const frames: Point[][] = [];
		for (let r = 0; r < reps; r++) {
			for (let i = 0; i <= 60; i++) {
				const b = (maxBend * (1 - Math.cos((i / 60) * 2 * Math.PI))) / 2;
				frames.push(skeleton({ kneeBend: b, lean: 10 + (lean - 10) * (b / Math.max(maxBend, 0.01)) }));
			}
		}
		return frames;
	}

	it('tiefe Kniebeugen: gezählt und sauber', () => {
		const a = new PoseAnalyzer('squat', { smoothing: 1 });
		const results = run(a, squatFrames(3, 1));
		expect(a.reps).toBe(3);
		expect(results.every((r) => r.cues[0].level === 'ok')).toBe(true);
	});

	it('halbe Kniebeugen: Hinweis „tiefer gehen“', () => {
		const a = new PoseAnalyzer('squat', { smoothing: 1 });
		const results = run(a, squatFrames(2, 0.55));
		expect(a.reps).toBe(2);
		expect(results.every((r) => r.cues.some((c) => c.id === 'depth'))).toBe(true);
	});

	it('starke Vorlage: Hinweis „Brust aufrecht“', () => {
		const a = new PoseAnalyzer('squat', { smoothing: 1 });
		const results = run(a, squatFrames(1, 1, 70));
		expect(results[0].cues.some((c) => c.id === 'lean')).toBe(true);
	});

	it('Press: Lockout erkannt vs. halber Druck', () => {
		const pressFrames = (top: number) => {
			const frames: Point[][] = [];
			for (let r = 0; r < 2; r++) {
				for (let i = 0; i <= 40; i++) {
					const u = (top * (1 - Math.cos((i / 40) * 2 * Math.PI))) / 2;
					frames.push(skeleton({ armUp: u }));
				}
			}
			return frames;
		};
		const full = new PoseAnalyzer('press', { smoothing: 1 });
		const r1 = run(full, pressFrames(1));
		expect(full.reps).toBe(2);
		expect(r1.every((r) => r.cues[0].level === 'ok')).toBe(true);

		const half = new PoseAnalyzer('press', { smoothing: 1 });
		const r2 = run(half, pressFrames(0.75));
		expect(half.reps).toBe(2);
		expect(r2.every((r) => r.cues.some((c) => c.id === 'lockout'))).toBe(true);
	});

	it('Swing: Lockout, unvollständige Streckung und „squatty swing“', () => {
		const swingFrames = (reps: number, topLean: number, bottomKnee: number) => {
			const frames: Point[][] = [];
			for (let r = 0; r < reps; r++) {
				for (let i = 0; i <= 45; i++) {
					const u = (1 - Math.cos((i / 45) * 2 * Math.PI)) / 2; // 0 oben, 1 unten
					frames.push(skeleton({ kneeBend: 0.05 + bottomKnee * u, lean: topLean + (75 - topLean) * u }));
				}
			}
			// kurz oben stehen bleiben, damit die letzte Wiederholung bewertet wird
			for (let i = 0; i < 30; i++) frames.push(skeleton({ kneeBend: 0.05, lean: topLean }));
			return frames;
		};
		const good = new PoseAnalyzer('swing', { smoothing: 1 });
		const r1 = run(good, swingFrames(3, 2, 0.25));
		expect(good.reps).toBe(3);
		expect(r1.every((r) => r.cues[0].level === 'ok')).toBe(true);

		const soft = new PoseAnalyzer('swing', { smoothing: 1 });
		const r2 = run(soft, swingFrames(2, 14, 0.25));
		expect(soft.reps).toBe(2);
		expect(r2.every((r) => r.cues.some((c) => c.id === 'lockout'))).toBe(true);

		const squatty = new PoseAnalyzer('swing', { smoothing: 1 });
		const r3 = run(squatty, swingFrames(2, 2, 0.75));
		expect(r3.every((r) => r.cues.some((c) => c.id === 'squatty'))).toBe(true);
	});

	it('ohne Profil: nur Landmarks, keine Zählung', () => {
		const a = new PoseAnalyzer(null);
		const out = a.process(skeleton({}), 0, 1);
		expect(out.landmarks).toHaveLength(33);
		expect(out.signal).toBeNull();
	});

	it('keine Person → leeres Ergebnis', () => {
		const a = new PoseAnalyzer('swing');
		expect(a.process(null, 0, 1).trackable).toBe(false);
	});
});

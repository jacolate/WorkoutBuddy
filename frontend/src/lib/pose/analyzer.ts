// Verbindet Landmarks → Glättung → Kennzahlen → Rep-Zähler → Bewertung.

import type { PoseProfileId } from '#lib/types.ts';
import { computeMetrics, type Metrics, type Point } from './geometry';
import { PROFILES, type Cue, type PoseProfile } from './profiles';
import { RepCounter, type RepSample } from './repCounter';

export interface RepResult {
	rep: number;
	cues: Cue[];
}

export interface AnalyzerOutput {
	landmarks: Point[];
	metrics: Metrics | null;
	/** Für das Profil nötige Körperteile sind sichtbar */
	trackable: boolean;
	signal: number | null;
	counted: boolean;
	result?: RepResult;
}

export class PoseAnalyzer {
	readonly profile: PoseProfile | null;
	#counter: RepCounter<Metrics> | null;
	#smoothed: Point[] | null = null;
	#alpha: number;
	results: RepResult[] = [];

	constructor(profileId: PoseProfileId | null, { smoothing = 0.55 } = {}) {
		this.profile = profileId ? PROFILES[profileId] : null;
		this.#counter = this.profile ? new RepCounter<Metrics>(this.profile.counter) : null;
		this.#alpha = smoothing;
	}

	get reps(): number {
		return this.#counter?.count ?? 0;
	}

	reset() {
		this.#counter?.reset();
		this.#smoothed = null;
		this.results = [];
	}

	/** Exponentielle Glättung der Landmarks gegen Zittern. */
	#smooth(lms: Point[]): Point[] {
		const a = this.#alpha;
		if (!this.#smoothed || this.#smoothed.length !== lms.length) {
			this.#smoothed = lms.map((p) => ({ ...p }));
			return this.#smoothed;
		}
		this.#smoothed = lms.map((p, i) => {
			const prev = this.#smoothed![i];
			return {
				x: prev.x + a * (p.x - prev.x),
				y: prev.y + a * (p.y - prev.y),
				visibility: p.visibility
			};
		});
		return this.#smoothed;
	}

	process(raw: Point[] | null, t: number, aspect: number): AnalyzerOutput {
		if (!raw || raw.length === 0) {
			this.#smoothed = null;
			return { landmarks: [], metrics: null, trackable: false, signal: null, counted: false };
		}
		const landmarks = this.#smooth(raw);
		const metrics = computeMetrics(landmarks, aspect);
		const out: AnalyzerOutput = { landmarks, metrics, trackable: !!metrics, signal: null, counted: false };
		if (!metrics || !this.profile || !this.#counter) return out;

		out.trackable = (!this.profile.needsLegs || metrics.legsVisible) && (!this.profile.needsArm || metrics.arm.visible);
		if (!out.trackable) return out;

		out.signal = this.profile.signal(metrics);
		const r = this.#counter.push({ t, signal: out.signal, data: metrics });
		out.counted = r.counted;
		if (r.completed) out.result = this.#evaluate(r.completed);
		return out;
	}

	/** Offene Wiederholung abschließen (z. B. wenn der Satz endet). */
	flush(): RepResult | undefined {
		const s = this.#counter?.flush();
		return s ? this.#evaluate(s) : undefined;
	}

	#evaluate(sample: RepSample<Metrics>): RepResult {
		const res = { rep: sample.rep, cues: this.profile!.evaluate(sample) };
		this.results.push(res);
		return res;
	}
}

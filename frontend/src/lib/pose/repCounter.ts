// Generischer Wiederholungszähler mit Hysterese.
//
// Ein Signal (z. B. Kniewinkel) pendelt zwischen einer Ruheposition und einer
// aktiven Position. Eine Wiederholung zählt, wenn das Signal die Ruheposition
// verlässt (activeThreshold überschreitet) und wieder zurückkehrt (restThreshold).
// Durch zwei getrennte Schwellen zählt Zittern um einen Wert nicht doppelt.

export interface RepCounterConfig {
	/** true: Ruhe = hoher Wert (z. B. Kniewinkel im Stand), false: Ruhe = niedriger Wert */
	restIsHigh: boolean;
	/** Ab diesem Wert gilt die Bewegung als begonnen */
	activeThreshold: number;
	/** Ab diesem Wert gilt die Wiederholung als abgeschlossen */
	restThreshold: number;
	minRepMs?: number;
	maxRepMs?: number;
	/** Wie lange nach der Rückkehr in Ruhe noch Frames gesammelt werden (für Lockout-Checks) */
	settleMs?: number;
}

export interface Frame<T> {
	t: number;
	signal: number;
	data: T;
}

export interface RepSample<T> {
	/** Frames der aktiven Phase */
	active: Frame<T>[];
	/** Frames direkt nach der Rückkehr in Ruhe */
	settle: Frame<T>[];
	rep: number;
}

export interface PushResult<T> {
	/** In diesem Frame wurde eine Wiederholung gezählt */
	counted: boolean;
	/** Eine Wiederholung ist vollständig erfasst und kann bewertet werden */
	completed?: RepSample<T>;
}

export class RepCounter<T> {
	count = 0;
	#cfg: Required<RepCounterConfig>;
	#state: 'rest' | 'active' = 'rest';
	#active: Frame<T>[] = [];
	#activeStart = 0;
	#pending: { sample: RepSample<T>; since: number } | null = null;

	constructor(cfg: RepCounterConfig) {
		this.#cfg = { minRepMs: 350, maxRepMs: 12_000, settleMs: 700, ...cfg };
	}

	get state() {
		return this.#state;
	}

	#isActive(v: number) {
		return this.#cfg.restIsHigh ? v < this.#cfg.activeThreshold : v > this.#cfg.activeThreshold;
	}

	#isRest(v: number) {
		return this.#cfg.restIsHigh ? v > this.#cfg.restThreshold : v < this.#cfg.restThreshold;
	}

	push(frame: Frame<T>): PushResult<T> {
		const res: PushResult<T> = { counted: false };

		if (this.#state === 'rest') {
			if (this.#pending) {
				if (this.#isActive(frame.signal) || frame.t - this.#pending.since >= this.#cfg.settleMs) {
					res.completed = this.#pending.sample;
					this.#pending = null;
				} else {
					this.#pending.sample.settle.push(frame);
				}
			}
			if (this.#isActive(frame.signal)) {
				this.#state = 'active';
				this.#active = [frame];
				this.#activeStart = frame.t;
			}
			return res;
		}

		// aktiv
		this.#active.push(frame);
		if (frame.t - this.#activeStart > this.#cfg.maxRepMs) {
			// Zu lange in Bewegung (z. B. Pause in der Hocke) → verwerfen.
			this.#state = 'rest';
			this.#active = [];
			return res;
		}
		if (this.#isRest(frame.signal)) {
			this.#state = 'rest';
			if (frame.t - this.#activeStart >= this.#cfg.minRepMs) {
				this.count += 1;
				res.counted = true;
				this.#pending = { sample: { active: this.#active, settle: [frame], rep: this.count }, since: frame.t };
			}
			this.#active = [];
		}
		return res;
	}

	/** Liefert eine noch offene Wiederholung (z. B. beim Stoppen). */
	flush(): RepSample<T> | undefined {
		const p = this.#pending?.sample;
		this.#pending = null;
		return p;
	}

	reset() {
		this.count = 0;
		this.#state = 'rest';
		this.#active = [];
		this.#pending = null;
	}
}

// Player-Engine: spielt die kompilierten Schritte ab. Die Zeit wird immer aus der
// Uhr berechnet (keine aufaddierten Intervalle) → kein Drift, und nach Bildschirm-aus
// oder Tab-Wechsel springt der Timer korrekt auf die tatsächliche Position.

import type { Step } from './compile';

export type EngineStatus = 'idle' | 'running' | 'paused' | 'finished';

export interface StepEndInfo {
	elapsedSec: number;
	skipped: boolean;
}

export interface EngineEvents {
	/** Neuer Schritt beginnt. catchUp = true, wenn Schritte im Hintergrund übersprungen wurden. */
	onStepStart?(step: Step, info: { catchUp: boolean }): void;
	onStepEnd?(step: Step, info: StepEndInfo): void;
	/** Ganzzahlige Restsekunden eines Timer-Schritts haben sich geändert. */
	onSecond?(remaining: number, step: Step): void;
	onFinish?(): void;
}

export class PlayerEngine {
	readonly steps: Step[];
	index = $state(0);
	status = $state<EngineStatus>('idle');
	/** Verstrichene Zeit im aktuellen Schritt (ms) */
	stepElapsedMs = $state(0);
	/** Gesamtzeit ohne Pausen (ms) */
	totalElapsedMs = $state(0);

	#events: EngineEvents;
	#now: () => number;
	#stepStart = 0; // Zeitpunkt (Uhr), an dem der aktuelle Schritt begann (pausenbereinigt)
	#startedAt = 0;
	#pausedAt = 0;
	#pausedTotal = 0;
	#lastSecond = -1;

	constructor(steps: Step[], events: EngineEvents = {}, now: () => number = () => performance.now()) {
		this.steps = steps;
		this.#events = events;
		this.#now = now;
	}

	get step(): Step | undefined {
		return this.steps[this.index];
	}

	get next(): Step | undefined {
		return this.steps[this.index + 1];
	}

	/** Nächster Arbeitsschritt (für „Als nächstes“ in Pausen). */
	get nextWork(): Step | undefined {
		for (let i = this.index + 1; i < this.steps.length; i++) {
			if (this.steps[i].kind === 'work') return this.steps[i];
		}
		return undefined;
	}

	get remainingSec(): number | null {
		const s = this.step;
		if (!s || s.advance !== 'timer' || s.durationSec === null) return null;
		return Math.max(0, s.durationSec - this.stepElapsedMs / 1000);
	}

	/** Fortschritt im aktuellen Timer-Schritt 0..1 */
	get stepProgress(): number {
		const s = this.step;
		if (!s || s.advance !== 'timer' || !s.durationSec) return 0;
		return Math.min(1, this.stepElapsedMs / 1000 / s.durationSec);
	}

	get overallProgress(): number {
		if (this.steps.length === 0) return 1;
		if (this.status === 'finished') return 1;
		return (this.index + this.stepProgress) / this.steps.length;
	}

	start() {
		if (this.status !== 'idle') return;
		if (this.steps.length === 0) {
			this.status = 'finished';
			this.#events.onFinish?.();
			return;
		}
		const now = this.#now();
		this.#startedAt = now;
		this.#stepStart = now;
		this.status = 'running';
		this.#lastSecond = -1;
		this.#events.onStepStart?.(this.steps[0], { catchUp: false });
		this.tick();
	}

	pause() {
		if (this.status !== 'running') return;
		this.tick();
		this.#pausedAt = this.#now();
		this.status = 'paused';
	}

	resume() {
		if (this.status !== 'paused') return;
		const pausedFor = this.#now() - this.#pausedAt;
		this.#stepStart += pausedFor;
		this.#pausedTotal += pausedFor;
		this.status = 'running';
		this.tick();
	}

	togglePause() {
		if (this.status === 'running') this.pause();
		else if (this.status === 'paused') this.resume();
	}

	/** Aktuellen Schritt als erledigt markieren (z. B. „Fertig“ bei Reps). */
	complete() {
		this.#advance(false);
	}

	/** Aktuellen Schritt überspringen. */
	skip() {
		this.#advance(true);
	}

	/** Zum vorherigen Schritt (bzw. aktuellen neu starten, wenn > 3 s gelaufen). */
	back() {
		if (this.status === 'idle' || this.status === 'finished') return;
		const now = this.status === 'paused' ? this.#pausedAt : this.#now();
		if (now - this.#stepStart > 3000 || this.index === 0) {
			this.#stepStart = now;
		} else {
			this.index -= 1;
			this.#stepStart = now;
		}
		this.#lastSecond = -1;
		this.stepElapsedMs = 0;
		this.#events.onStepStart?.(this.steps[this.index], { catchUp: false });
	}

	/** Muss regelmäßig aufgerufen werden (rAF/Intervall). */
	tick() {
		if (this.status !== 'running') return;
		const now = this.#now();
		this.totalElapsedMs = now - this.#startedAt - this.#pausedTotal;
		let elapsed = now - this.#stepStart;

		// Abgelaufene Timer-Schritte weiterschalten (ggf. mehrere nach langer Hintergrundzeit).
		for (;;) {
			const s = this.steps[this.index];
			if (!s || s.advance !== 'timer' || s.durationSec === null) break;
			const durMs = s.durationSec * 1000;
			if (elapsed < durMs) break;
			this.#events.onStepEnd?.(s, { elapsedSec: s.durationSec, skipped: false });
			this.#stepStart += durMs;
			elapsed -= durMs;
			this.index += 1;
			this.#lastSecond = -1;
			if (this.index >= this.steps.length) {
				this.#finish();
				return;
			}
			const more = this.#willAdvanceAgain(elapsed);
			this.#events.onStepStart?.(this.steps[this.index], { catchUp: more });
		}

		this.stepElapsedMs = elapsed;
		const rem = this.remainingSec;
		if (rem !== null) {
			const sec = Math.ceil(rem);
			if (sec !== this.#lastSecond) {
				this.#lastSecond = sec;
				this.#events.onSecond?.(sec, this.steps[this.index]);
			}
		}
	}

	#willAdvanceAgain(elapsedMs: number): boolean {
		const s = this.steps[this.index];
		return !!s && s.advance === 'timer' && s.durationSec !== null && elapsedMs >= s.durationSec * 1000;
	}

	#advance(skipped: boolean) {
		if (this.status !== 'running' && this.status !== 'paused') return;
		const now = this.status === 'paused' ? this.#pausedAt : this.#now();
		const s = this.steps[this.index];
		const elapsedSec = Math.max(0, (now - this.#stepStart) / 1000);
		this.#events.onStepEnd?.(s, { elapsedSec, skipped });
		this.index += 1;
		this.#stepStart = now;
		this.#lastSecond = -1;
		this.stepElapsedMs = 0;
		if (this.index >= this.steps.length) {
			this.#finish();
			return;
		}
		this.#events.onStepStart?.(this.steps[this.index], { catchUp: false });
		if (this.status === 'running') this.tick();
	}

	#finish() {
		this.index = this.steps.length;
		this.status = 'finished';
		this.stepElapsedMs = 0;
		this.#events.onFinish?.();
	}
}

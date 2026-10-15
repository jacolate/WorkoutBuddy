// Übungsprofile: welches Signal zählt Wiederholungen, und welche einfachen
// Heuristiken prüfen die Ausführung. Alle Schwellwerte stehen gebündelt in
// THRESHOLDS, damit sie nach ein paar echten Trainings leicht feinjustiert werden können.
//
// Wichtig: Das sind grobe Hinweise aus 2D-Kamerabildern, keine exakte Bewegungsanalyse.

import type { PoseProfileId } from '#lib/types.ts';
import type { Metrics } from './geometry';
import type { RepCounterConfig, RepSample } from './repCounter';

export interface Cue {
	id: string;
	level: 'ok' | 'warn';
	text: string;
}

export interface PoseProfile {
	id: PoseProfileId;
	name: string;
	cameraHint: string;
	/** Benötigt sichtbare Beine (Seitenansicht) */
	needsLegs: boolean;
	/** Benötigt sichtbaren Arm */
	needsArm: boolean;
	signal(m: Metrics): number;
	signalName: string;
	counter: RepCounterConfig;
	evaluate(rep: RepSample<Metrics>): Cue[];
}

export const THRESHOLDS = {
	squat: {
		activeKnee: 125, // darunter: Abwärtsbewegung begonnen
		restKnee: 155, // darüber: wieder im Stand
		minHipBelowKnee: -0.12, // Hüfte max. ~12 % Oberschenkellänge über Knie
		maxTorsoLean: 55
	},
	swing: {
		activeHip: 125,
		restHip: 155,
		lockoutHip: 165, // volle Hüftstreckung oben
		minKneeAtBottom: 115 // kleiner = zu viel Kniebeuge („squatty swing“)
	},
	hinge: {
		activeHip: 120,
		restHip: 155,
		lockoutHip: 165,
		minKneeAtBottom: 105
	},
	press: {
		activeElbow: 140,
		restElbow: 100,
		lockoutElbow: 160,
		minWristAboveShoulder: 0.55, // Handgelenk deutlich über Schulterhöhe (rel. Rumpflänge)
		maxArmFromVertical: 25
	}
};

const max = (xs: number[]) => xs.reduce((a, b) => Math.max(a, b), -Infinity);
const min = (xs: number[]) => xs.reduce((a, b) => Math.min(a, b), Infinity);

function okIfEmpty(cues: Cue[]): Cue[] {
	return cues.length > 0 ? cues : [{ id: 'clean', level: 'ok', text: 'Saubere Wiederholung' }];
}

function all(rep: RepSample<Metrics>) {
	return [...rep.active, ...rep.settle].map((f) => f.data);
}

const squat: PoseProfile = {
	id: 'squat',
	name: 'Kniebeuge',
	cameraHint: 'Handy seitlich aufstellen, ganzer Körper im Bild',
	needsLegs: true,
	needsArm: false,
	signal: (m) => m.kneeAngle,
	signalName: 'Kniewinkel',
	counter: { restIsHigh: true, activeThreshold: THRESHOLDS.squat.activeKnee, restThreshold: THRESHOLDS.squat.restKnee },
	evaluate(rep) {
		const t = THRESHOLDS.squat;
		const frames = rep.active.map((f) => f.data);
		const cues: Cue[] = [];
		if (max(frames.map((m) => m.hipBelowKnee)) < t.minHipBelowKnee) {
			cues.push({ id: 'depth', level: 'warn', text: 'Tiefer gehen – Hüfte mind. auf Kniehöhe' });
		}
		// Rumpfneigung am tiefsten Punkt
		const bottom = frames.reduce((a, b) => (b.kneeAngle < a.kneeAngle ? b : a), frames[0]);
		if (bottom && bottom.torsoLean > t.maxTorsoLean) {
			cues.push({ id: 'lean', level: 'warn', text: 'Brust aufrecht – Oberkörper kippt nach vorn' });
		}
		return okIfEmpty(cues);
	}
};

const swing: PoseProfile = {
	id: 'swing',
	name: 'Swing',
	cameraHint: 'Handy seitlich aufstellen, ganzer Körper im Bild',
	needsLegs: true,
	needsArm: false,
	signal: (m) => m.hipAngle,
	signalName: 'Hüftwinkel',
	counter: { restIsHigh: true, activeThreshold: THRESHOLDS.swing.activeHip, restThreshold: THRESHOLDS.swing.restHip },
	evaluate(rep) {
		const t = THRESHOLDS.swing;
		const cues: Cue[] = [];
		if (max(all(rep).map((m) => m.hipAngle)) < t.lockoutHip) {
			cues.push({ id: 'lockout', level: 'warn', text: 'Oben Hüfte voll strecken – Po anspannen' });
		}
		if (min(rep.active.map((f) => f.data.kneeAngle)) < t.minKneeAtBottom) {
			cues.push({ id: 'squatty', level: 'warn', text: 'Mehr Hüfte, weniger Knie – Hinge statt Squat' });
		}
		return okIfEmpty(cues);
	}
};

const hinge: PoseProfile = {
	id: 'hinge',
	name: 'Hinge / Kreuzheben',
	cameraHint: 'Handy seitlich aufstellen, ganzer Körper im Bild',
	needsLegs: true,
	needsArm: false,
	signal: (m) => m.hipAngle,
	signalName: 'Hüftwinkel',
	counter: { restIsHigh: true, activeThreshold: THRESHOLDS.hinge.activeHip, restThreshold: THRESHOLDS.hinge.restHip },
	evaluate(rep) {
		const t = THRESHOLDS.hinge;
		const cues: Cue[] = [];
		if (max(all(rep).map((m) => m.hipAngle)) < t.lockoutHip) {
			cues.push({ id: 'lockout', level: 'warn', text: 'Oben ganz aufrichten' });
		}
		if (min(rep.active.map((f) => f.data.kneeAngle)) < t.minKneeAtBottom) {
			cues.push({ id: 'squatty', level: 'warn', text: 'Hüfte nach hinten schieben statt Knie stark beugen' });
		}
		return okIfEmpty(cues);
	}
};

const press: PoseProfile = {
	id: 'press',
	name: 'Überkopfdrücken',
	cameraHint: 'Handy frontal aufstellen, Oberkörper und Arme im Bild',
	needsLegs: false,
	needsArm: true,
	signal: (m) => m.arm.elbowAngle,
	signalName: 'Ellbogenwinkel',
	counter: { restIsHigh: false, activeThreshold: THRESHOLDS.press.activeElbow, restThreshold: THRESHOLDS.press.restElbow },
	evaluate(rep) {
		const t = THRESHOLDS.press;
		const frames = rep.active.map((f) => f.data);
		const cues: Cue[] = [];
		const top = frames.reduce((a, b) => (b.arm.elbowAngle > a.arm.elbowAngle ? b : a), frames[0]);
		if (!top) return okIfEmpty(cues);
		if (top.arm.elbowAngle < t.lockoutElbow) {
			cues.push({ id: 'lockout', level: 'warn', text: 'Arm oben ganz durchstrecken' });
		}
		if (max(frames.map((m) => m.arm.wristAboveShoulder)) < t.minWristAboveShoulder) {
			cues.push({ id: 'height', level: 'warn', text: 'Gewicht ganz über den Kopf drücken' });
		}
		if (top.arm.armFromVertical > t.maxArmFromVertical) {
			cues.push({ id: 'vertical', level: 'warn', text: 'Arm senkrecht – Bizeps neben das Ohr' });
		}
		return okIfEmpty(cues);
	}
};

export const PROFILES: Record<PoseProfileId, PoseProfile> = { squat, swing, hinge, press };

export const PROFILE_OPTIONS: { id: PoseProfileId; label: string }[] = [
	{ id: 'swing', label: 'Swing' },
	{ id: 'squat', label: 'Kniebeuge' },
	{ id: 'hinge', label: 'Hinge / Kreuzheben' },
	{ id: 'press', label: 'Überkopfdrücken' }
];

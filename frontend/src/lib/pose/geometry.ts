// Gelenkwinkel und Kennzahlen aus MediaPipe-Pose-Landmarks.

export interface Point {
	x: number;
	y: number;
	visibility?: number;
}

/** MediaPipe BlazePose Landmark-Indizes */
export const LM = {
	nose: 0,
	leftShoulder: 11,
	rightShoulder: 12,
	leftElbow: 13,
	rightElbow: 14,
	leftWrist: 15,
	rightWrist: 16,
	leftHip: 23,
	rightHip: 24,
	leftKnee: 25,
	rightKnee: 26,
	leftAnkle: 27,
	rightAnkle: 28
} as const;

/** Linien für das Skelett-Overlay (ohne Gesicht/Hände). */
export const SKELETON: [number, number][] = [
	[11, 12],
	[11, 13],
	[13, 15],
	[12, 14],
	[14, 16],
	[11, 23],
	[12, 24],
	[23, 24],
	[23, 25],
	[25, 27],
	[24, 26],
	[26, 28],
	[27, 31],
	[28, 32],
	[27, 29],
	[28, 30]
];

export const MIN_VISIBILITY = 0.5;

/** Winkel ABC am Punkt B in Grad (0–180). */
export function angle(a: Point, b: Point, c: Point): number {
	const abx = a.x - b.x;
	const aby = a.y - b.y;
	const cbx = c.x - b.x;
	const cby = c.y - b.y;
	const dot = abx * cbx + aby * cby;
	const mag = Math.hypot(abx, aby) * Math.hypot(cbx, cby);
	if (mag === 0) return 180;
	const cos = Math.min(1, Math.max(-1, dot / mag));
	return (Math.acos(cos) * 180) / Math.PI;
}

/** Winkel der Strecke from→to zur Senkrechten (0 = senkrecht), unabhängig von der Richtung. */
export function angleFromVertical(from: Point, to: Point): number {
	const dx = to.x - from.x;
	const dy = to.y - from.y;
	if (dx === 0 && dy === 0) return 0;
	const deg = (Math.atan2(Math.abs(dx), Math.abs(dy)) * 180) / Math.PI;
	return deg;
}

function dist(a: Point, b: Point): number {
	return Math.hypot(a.x - b.x, a.y - b.y);
}

const vis = (p: Point | undefined) => p?.visibility ?? 1;

export interface ArmMetrics {
	elbowAngle: number;
	/** (Schulter.y − Handgelenk.y) / Rumpflänge; > 0 = Handgelenk über Schulter */
	wristAboveShoulder: number;
	/** Winkel Schulter→Handgelenk zur Senkrechten */
	armFromVertical: number;
	visible: boolean;
}

export interface Metrics {
	/** Körperseite, die für Bein-/Hüftwinkel verwendet wurde (bessere Sichtbarkeit) */
	legSide: 'left' | 'right';
	legsVisible: boolean;
	kneeAngle: number;
	hipAngle: number;
	/** Rumpfneigung zur Senkrechten (0 = aufrecht) */
	torsoLean: number;
	/** (Hüfte.y − Knie.y) / Oberschenkellänge; ≥ 0 = Hüfte auf/unter Kniehöhe */
	hipBelowKnee: number;
	/** Arm mit dem höheren Handgelenk (Druck-Arm bei einarmigen Übungen) */
	arm: ArmMetrics;
	armSide: 'left' | 'right';
}

/**
 * Berechnet Kennzahlen. `aspect` = Breite/Höhe des Videos, damit Winkel in
 * echten Proportionen gerechnet werden (Landmarks sind auf 0..1 normiert).
 */
export function computeMetrics(lms: Point[], aspect: number): Metrics | null {
	if (lms.length < 29) return null;
	const p = (i: number): Point => ({ x: lms[i].x * aspect, y: lms[i].y, visibility: lms[i].visibility });

	const sideScore = (s: 'left' | 'right') => {
		const idx = s === 'left' ? [LM.leftShoulder, LM.leftHip, LM.leftKnee, LM.leftAnkle] : [LM.rightShoulder, LM.rightHip, LM.rightKnee, LM.rightAnkle];
		return idx.reduce((acc, i) => acc + vis(lms[i]), 0) / idx.length;
	};
	const legSide = sideScore('left') >= sideScore('right') ? 'left' : 'right';
	const L = legSide === 'left';
	const shoulder = p(L ? LM.leftShoulder : LM.rightShoulder);
	const hip = p(L ? LM.leftHip : LM.rightHip);
	const knee = p(L ? LM.leftKnee : LM.rightKnee);
	const ankle = p(L ? LM.leftAnkle : LM.rightAnkle);
	const legsVisible = [shoulder, hip, knee, ankle].every((pt) => vis(pt) >= MIN_VISIBILITY);

	const thigh = dist(hip, knee) || 1;
	const torso = dist(shoulder, hip) || 1;

	const armFor = (s: 'left' | 'right'): ArmMetrics => {
		const sh = p(s === 'left' ? LM.leftShoulder : LM.rightShoulder);
		const el = p(s === 'left' ? LM.leftElbow : LM.rightElbow);
		const wr = p(s === 'left' ? LM.leftWrist : LM.rightWrist);
		return {
			elbowAngle: angle(sh, el, wr),
			wristAboveShoulder: (sh.y - wr.y) / torso,
			armFromVertical: angleFromVertical(sh, wr),
			visible: [sh, el, wr].every((pt) => vis(pt) >= MIN_VISIBILITY)
		};
	};
	const left = armFor('left');
	const right = armFor('right');
	let armSide: 'left' | 'right';
	if (left.visible !== right.visible) armSide = left.visible ? 'left' : 'right';
	else armSide = left.wristAboveShoulder >= right.wristAboveShoulder ? 'left' : 'right';

	return {
		legSide,
		legsVisible,
		kneeAngle: angle(hip, knee, ankle),
		hipAngle: angle(shoulder, hip, knee),
		torsoLean: angleFromVertical(hip, shoulder),
		hipBelowKnee: (hip.y - knee.y) / thigh,
		arm: armSide === 'left' ? left : right,
		armSide
	};
}

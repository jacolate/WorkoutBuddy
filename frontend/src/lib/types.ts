// Gemeinsame Typen für API und UI. Die Workout-Struktur wird vom Backend als JSON
// unverändert gespeichert – dieses Schema ist die Quelle der Wahrheit.

export type PoseProfileId = 'swing' | 'squat' | 'hinge' | 'press';

export interface Exercise {
	id: number;
	name: string;
	description: string;
	category: string;
	unilateral: boolean;
	defaultWeightKg: number | null;
	poseProfile: PoseProfileId | null;
	cues: string;
	cameraHint: string;
	createdAt?: string;
	updatedAt?: string;
}

/** Gewicht in kg, 'bw' = Körpergewicht, null = nicht festgelegt. */
export type Weight = number | 'bw' | null;

export interface BlockItem {
	exerciseId: number;
	reps?: number | null;
	durationSec?: number | null;
	weight: Weight;
}

interface BlockBase {
	id: string;
	title?: string;
	/** Arbeitsphasen dieses Blocks automatisch aufnehmen (wenn Kamera aktiv). */
	record?: boolean;
}

export interface SetsBlock extends BlockBase {
	type: 'sets';
	item: BlockItem;
	sets: number;
	restSec: number;
}

export interface CircuitBlock extends BlockBase {
	type: 'circuit';
	rounds: number;
	items: BlockItem[];
	restBetweenItemsSec: number;
	restBetweenRoundsSec: number;
}

export interface IntervalBlock extends BlockBase {
	type: 'interval';
	rounds: number;
	workSec: number;
	restSec: number;
	items: BlockItem[];
}

export interface EmomBlock extends BlockBase {
	type: 'emom';
	minutes: number;
	items: BlockItem[];
}

export interface AmrapBlock extends BlockBase {
	type: 'amrap';
	durationSec: number;
	items: BlockItem[];
}

export interface RestBlock extends BlockBase {
	type: 'rest';
	durationSec: number;
}

export type Block = SetsBlock | CircuitBlock | IntervalBlock | EmomBlock | AmrapBlock | RestBlock;
export type BlockType = Block['type'];

export interface WorkoutStructure {
	version: 1;
	prepareSec: number;
	blockRestSec: number;
	blocks: Block[];
}

export interface Workout {
	id: number;
	name: string;
	notes: string;
	structure: WorkoutStructure;
	createdAt: string;
	updatedAt: string;
	lastPlayedAt?: string;
}

export type Side = 'L' | 'R';

/** Ein Eintrag im Session-Log (pro abgeschlossenem Arbeitsschritt). */
export interface LogEntry {
	step: number;
	blockIndex: number;
	exerciseId: number | null;
	exerciseName: string;
	weight: Weight;
	side?: Side;
	reps: number | null;
	/** AMRAP: abgeschlossene Runden */
	rounds?: number;
	durationSec: number;
	skipped?: boolean;
	at: string;
}

export interface Session {
	id: number;
	workoutId: number | null;
	workoutName: string;
	startedAt: string;
	finishedAt: string | null;
	snapshot: WorkoutStructure;
	log: LogEntry[];
	notes: string;
	recordingCount: number;
}

export interface Recording {
	id: number;
	sessionId: number | null;
	exerciseId: number | null;
	exerciseName: string | null;
	poseProfile: PoseProfileId | null;
	mime: string;
	sizeBytes: number;
	durationSec: number;
	label: string;
	note: string;
	createdAt: string;
}

export interface Settings {
	weights: number[];
	sound: boolean;
	voice: boolean;
	prepareSec: number;
	secondsPerRep: number;
	/** Rep-Schritt automatisch beenden, wenn der Pose-Zähler das Ziel erreicht. */
	autoAdvanceReps: boolean;
}

export const DEFAULT_SETTINGS: Settings = {
	weights: [12, 16],
	sound: true,
	voice: true,
	prepareSec: 10,
	secondsPerRep: 3,
	autoAdvanceReps: false
};

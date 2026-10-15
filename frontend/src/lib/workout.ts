// Hilfen rund um die Workout-Struktur (Beschreibung, neue Blöcke).
import type { Block, BlockItem, BlockType, Exercise, WorkoutStructure } from '#lib/types.ts';
import { blockLabel, compile, estimateDuration } from '#lib/timer/compile.ts';
import { formatDuration, formatWeightShort } from '#lib/format.ts';

export function uid(): string {
	return Math.random().toString(36).slice(2, 10);
}

export function emptyStructure(prepareSec = 10): WorkoutStructure {
	return { version: 1, prepareSec, blockRestSec: 60, blocks: [] };
}

function itemFor(e: Exercise | undefined, reps: number | null = 10): BlockItem {
	return {
		exerciseId: e?.id ?? 0,
		reps,
		durationSec: null,
		weight: e?.defaultWeightKg ?? (e ? 'bw' : null)
	};
}

export function newBlock(type: BlockType, first?: Exercise): Block {
	const id = uid();
	switch (type) {
		case 'sets':
			return { id, type, sets: 3, restSec: 60, item: itemFor(first, 10) };
		case 'circuit':
			return { id, type, rounds: 3, restBetweenItemsSec: 0, restBetweenRoundsSec: 60, items: first ? [itemFor(first, 10)] : [] };
		case 'interval':
			return { id, type, rounds: 8, workSec: 20, restSec: 10, items: first ? [itemFor(first, null)] : [] };
		case 'emom':
			return { id, type, minutes: 10, items: first ? [itemFor(first, 8)] : [] };
		case 'amrap':
			return { id, type, durationSec: 600, items: first ? [itemFor(first, 10)] : [] };
		case 'rest':
			return { id, type, durationSec: 120 };
	}
}

export function newItem(e: Exercise, block: Block): BlockItem {
	return itemFor(e, block.type === 'interval' ? null : 10);
}

function itemText(it: BlockItem, ex: Map<number, Exercise>, withAmount = true): string {
	const name = ex.get(it.exerciseId)?.name ?? '?';
	const amount = it.reps ? `${it.reps}×` : it.distanceM ? `${it.distanceM} m` : it.durationSec ? `${it.durationSec}s` : '';
	const w = it.weight !== null ? ` (${formatWeightShort(it.weight)})` : '';
	return `${withAmount && amount ? amount + ' ' : ''}${name}${w}`;
}

/** Einzeilige Beschreibung eines Blocks. */
export function describeBlock(b: Block, ex: Map<number, Exercise>): string {
	switch (b.type) {
		case 'sets': {
			const amount = b.item.reps ? `${b.item.reps}` : b.item.distanceM ? `${b.item.distanceM} m` : b.item.durationSec ? `${b.item.durationSec}s` : '?';
			return `${b.sets} × ${amount} ${itemText(b.item, ex, false)}`;
		}
		case 'circuit':
			return `${b.rounds} Runden: ${b.items.map((i) => itemText(i, ex)).join(', ') || '–'}`;
		case 'interval':
			return `${b.rounds} × ${b.workSec}/${b.restSec}s: ${b.items.map((i) => itemText(i, ex, false)).join(', ') || '–'}`;
		case 'emom':
			return `${b.minutes} min: ${b.items.map((i) => itemText(i, ex)).join(' → ') || '–'}`;
		case 'amrap':
			return `${formatDuration(b.durationSec)}: ${b.items.map((i) => itemText(i, ex)).join(', ') || '–'}`;
		case 'rest':
			return formatDuration(b.durationSec);
	}
}

export function workoutSummary(s: WorkoutStructure, ex: Map<number, Exercise>, secondsPerRep = 3) {
	const steps = compile(s, ex);
	return {
		steps,
		durationSec: estimateDuration(steps, secondsPerRep),
		blocks: s.blocks.length,
		labels: s.blocks.map((b) => blockLabel(b))
	};
}

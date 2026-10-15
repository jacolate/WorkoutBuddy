export const CATEGORIES: { id: string; label: string }[] = [
	{ id: 'swing', label: 'Swing & Ballistik' },
	{ id: 'squat', label: 'Kniebeuge' },
	{ id: 'hinge', label: 'Hinge' },
	{ id: 'press', label: 'Drücken' },
	{ id: 'pull', label: 'Ziehen' },
	{ id: 'clean', label: 'Clean' },
	{ id: 'lunge', label: 'Ausfallschritt' },
	{ id: 'carry', label: 'Tragen' },
	{ id: 'core', label: 'Rumpf' },
	{ id: 'mobility', label: 'Mobilität' },
	{ id: 'bodyweight', label: 'Körpergewicht' },
	{ id: 'other', label: 'Sonstiges' }
];

export function categoryLabel(id: string): string {
	return CATEGORIES.find((c) => c.id === id)?.label ?? 'Sonstiges';
}

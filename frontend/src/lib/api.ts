import { goto } from '$app/navigation';
import type { Exercise, LogEntry, Recording, Session, Settings, Workout, WorkoutStructure } from '#lib/types.ts';

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
	}
}

async function request<T>(method: string, path: string, body?: unknown, opts: { redirectOn401?: boolean } = {}): Promise<T> {
	const init: RequestInit = { method, credentials: 'same-origin', headers: {} };
	if (body instanceof FormData) {
		init.body = body;
	} else if (body !== undefined) {
		init.body = JSON.stringify(body);
		(init.headers as Record<string, string>)['Content-Type'] = 'application/json';
	}
	const res = await fetch(`/api${path}`, init);
	if (res.status === 401 && opts.redirectOn401 !== false) {
		const next = location.pathname + location.search;
		if (!location.pathname.startsWith('/login')) {
			await goto(`/login?next=${encodeURIComponent(next)}`, { replaceState: true });
		}
		throw new ApiError(401, 'Nicht angemeldet');
	}
	if (!res.ok) {
		let msg = res.statusText;
		try {
			msg = (await res.json()).error ?? msg;
		} catch {
			/* kein JSON */
		}
		throw new ApiError(res.status, msg);
	}
	if (res.status === 204) return undefined as T;
	return (await res.json()) as T;
}

export const api = {
	// Auth
	login: (username: string, password: string) =>
		request<{ user: string }>('POST', '/auth/login', { username, password }, { redirectOn401: false }),
	logout: () => request<void>('POST', '/auth/logout', undefined, { redirectOn401: false }),
	me: () => request<{ user: string }>('GET', '/auth/me', undefined, { redirectOn401: false }),

	// Übungen
	exercises: () => request<Exercise[]>('GET', '/exercises'),
	createExercise: (e: Partial<Exercise>) => request<Exercise>('POST', '/exercises', e),
	updateExercise: (e: Exercise) => request<Exercise>('PUT', `/exercises/${e.id}`, e),
	deleteExercise: (id: number) => request<void>('DELETE', `/exercises/${id}`),

	// Workouts
	workouts: () => request<Workout[]>('GET', '/workouts'),
	workout: (id: number) => request<Workout>('GET', `/workouts/${id}`),
	createWorkout: (w: { name: string; notes: string; structure: WorkoutStructure }) => request<Workout>('POST', '/workouts', w),
	updateWorkout: (w: { id: number; name: string; notes: string; structure: WorkoutStructure }) => request<Workout>('PUT', `/workouts/${w.id}`, w),
	deleteWorkout: (id: number) => request<void>('DELETE', `/workouts/${id}`),
	duplicateWorkout: (id: number) => request<Workout>('POST', `/workouts/${id}/duplicate`),

	// Sessions
	sessions: () => request<Session[]>('GET', '/sessions'),
	session: (id: number) => request<Session>('GET', `/sessions/${id}`),
	createSession: (s: { workoutId: number | null; workoutName: string; snapshot: WorkoutStructure; startedAt: string }) =>
		request<Session>('POST', '/sessions', s),
	patchSession: (id: number, p: { finishedAt?: string; log?: LogEntry[]; notes?: string }) => request<Session>('PATCH', `/sessions/${id}`, p),
	deleteSession: (id: number) => request<void>('DELETE', `/sessions/${id}`),

	// Aufnahmen
	recordings: (sessionId?: number) => request<Recording[]>('GET', `/recordings${sessionId ? `?sessionId=${sessionId}` : ''}`),
	recording: (id: number) => request<Recording>('GET', `/recordings/${id}`),
	uploadRecording: (form: FormData) => request<Recording>('POST', '/recordings', form),
	patchRecording: (id: number, note: string) => request<Recording>('PATCH', `/recordings/${id}`, { note }),
	deleteRecording: (id: number) => request<void>('DELETE', `/recordings/${id}`),
	recordingUrl: (id: number) => `/api/recordings/${id}/file`,

	// Einstellungen
	settings: () => request<Partial<Settings>>('GET', '/settings'),
	putSettings: (s: Partial<Settings>) => request<Partial<Settings>>('PUT', '/settings', s)
};

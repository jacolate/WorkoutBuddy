// Globaler App-Zustand: Übungsbibliothek, Einstellungen, Theme.
import { api } from '#lib/api.ts';
import { DEFAULT_SETTINGS, type Exercise, type Settings } from '#lib/types.ts';

class AppState {
	exercises = $state<Exercise[]>([]);
	settings = $state<Settings>({ ...DEFAULT_SETTINGS });
	user = $state<string | null>(null);
	loaded = $state(false);

	exerciseMap = $derived(new Map(this.exercises.map((e) => [e.id, e])));

	async load() {
		const [ex, st] = await Promise.all([api.exercises(), api.settings()]);
		this.exercises = ex;
		this.settings = { ...DEFAULT_SETTINGS, ...st };
		this.loaded = true;
	}

	async reloadExercises() {
		this.exercises = await api.exercises();
	}

	async saveSettings(patch: Partial<Settings>) {
		this.settings = { ...this.settings, ...patch };
		const st = await api.putSettings(patch);
		this.settings = { ...DEFAULT_SETTINGS, ...st };
	}

	reset() {
		this.exercises = [];
		this.settings = { ...DEFAULT_SETTINGS };
		this.user = null;
		this.loaded = false;
	}
}

export const app = new AppState();

export type ThemePref = 'system' | 'light' | 'dark';

class ThemeState {
	pref = $state<ThemePref>('system');
	#media: MediaQueryList | null = null;

	init() {
		try {
			this.pref = (localStorage.getItem('theme') as ThemePref) || 'system';
		} catch {
			this.pref = 'system';
		}
		this.#media = window.matchMedia('(prefers-color-scheme: dark)');
		this.#media.addEventListener('change', () => this.apply());
		this.apply();
	}

	set(pref: ThemePref) {
		this.pref = pref;
		try {
			localStorage.setItem('theme', pref);
		} catch {
			/* privater Modus */
		}
		this.apply();
	}

	apply() {
		const dark = this.pref === 'dark' || (this.pref === 'system' && !!this.#media?.matches);
		document.documentElement.dataset.theme = dark ? 'dark' : 'light';
	}
}

export const theme = new ThemeState();

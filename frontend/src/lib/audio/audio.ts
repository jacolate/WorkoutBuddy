// Töne (WebAudio) und Sprachansagen (SpeechSynthesis, de-DE).

let ctx: AudioContext | null = null;

/** Muss aus einer Nutzergeste (Tap auf „Start“) aufgerufen werden, sonst blockiert iOS den Ton. */
export function unlockAudio() {
	if (!ctx) {
		const Ctor = window.AudioContext ?? (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
		if (!Ctor) return;
		ctx = new Ctor();
	}
	if (ctx.state === 'suspended') ctx.resume();
	// Ein stummer Ton entsperrt die Wiedergabe auf iOS zuverlässig.
	tone(1, 0.01, 0);
	// Sprachausgabe ebenfalls „aufwecken“.
	if ('speechSynthesis' in window) {
		const u = new SpeechSynthesisUtterance('');
		u.volume = 0;
		speechSynthesis.speak(u);
	}
}

function tone(freq: number, durationSec: number, volume = 0.25, type: OscillatorType = 'sine') {
	if (!ctx) return;
	const t = ctx.currentTime;
	const osc = ctx.createOscillator();
	const gain = ctx.createGain();
	osc.type = type;
	osc.frequency.value = freq;
	gain.gain.setValueAtTime(0, t);
	gain.gain.linearRampToValueAtTime(volume, t + 0.01);
	gain.gain.exponentialRampToValueAtTime(0.0001, t + durationSec);
	osc.connect(gain).connect(ctx.destination);
	osc.start(t);
	osc.stop(t + durationSec + 0.02);
}

export const beeps = {
	/** 3-2-1 Countdown */
	tick: () => tone(880, 0.12, 0.3),
	/** Start einer Arbeitsphase */
	go: () => {
		tone(1320, 0.35, 0.35);
	},
	/** Start einer Pause */
	rest: () => {
		tone(660, 0.18, 0.3);
		setTimeout(() => tone(440, 0.3, 0.3), 160);
	},
	/** Workout fertig */
	done: () => {
		[660, 880, 1100, 1320].forEach((f, i) => setTimeout(() => tone(f, 0.25, 0.3), i * 140));
	},
	/** Wiederholung erkannt (leise) */
	rep: () => tone(1000, 0.06, 0.12, 'triangle')
};

let germanVoice: SpeechSynthesisVoice | null | undefined;

function pickVoice(): SpeechSynthesisVoice | null {
	if (germanVoice !== undefined) return germanVoice;
	const voices = speechSynthesis.getVoices();
	if (voices.length === 0) return null; // noch nicht geladen – später erneut versuchen
	germanVoice = voices.find((v) => v.lang === 'de-DE' && v.localService) ?? voices.find((v) => v.lang.startsWith('de')) ?? null;
	return germanVoice;
}

export function speak(text: string, { interrupt = true } = {}) {
	if (!('speechSynthesis' in window) || !text) return;
	if (interrupt) speechSynthesis.cancel();
	const u = new SpeechSynthesisUtterance(text);
	u.lang = 'de-DE';
	const v = pickVoice();
	if (v) u.voice = v;
	u.rate = 1.05;
	speechSynthesis.speak(u);
}

export function stopSpeaking() {
	if ('speechSynthesis' in window) speechSynthesis.cancel();
}

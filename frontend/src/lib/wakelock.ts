// Hält den Bildschirm während des Trainings an (nur in sicheren Kontexten verfügbar).

let sentinel: WakeLockSentinel | null = null;
let wanted = false;

async function request() {
	if (!('wakeLock' in navigator) || !wanted) return;
	try {
		sentinel = await navigator.wakeLock.request('screen');
		sentinel.addEventListener('release', () => {
			sentinel = null;
		});
	} catch {
		/* z. B. Energiesparmodus oder kein HTTPS */
	}
}

function onVisibility() {
	if (document.visibilityState === 'visible' && wanted && !sentinel) void request();
}

export function wakeLockSupported(): boolean {
	return 'wakeLock' in navigator && window.isSecureContext;
}

export async function keepAwake() {
	wanted = true;
	document.addEventListener('visibilitychange', onVisibility);
	await request();
}

export async function releaseAwake() {
	wanted = false;
	document.removeEventListener('visibilitychange', onVisibility);
	await sentinel?.release().catch(() => {});
	sentinel = null;
}

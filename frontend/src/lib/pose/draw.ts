import { MIN_VISIBILITY, SKELETON, type Point } from './geometry';

/**
 * Zeichnet das Skelett auf ein Canvas, das exakt über dem Video liegt
 * (gleiche CSS-Größe, object-fit: cover wird berücksichtigt).
 */
export function drawSkeleton(
	canvas: HTMLCanvasElement,
	video: HTMLVideoElement,
	landmarks: Point[],
	opts: { color: string; jointColor: string; fit?: 'cover' | 'contain' }
) {
	const dpr = window.devicePixelRatio || 1;
	const cw = canvas.clientWidth;
	const ch = canvas.clientHeight;
	if (canvas.width !== Math.round(cw * dpr) || canvas.height !== Math.round(ch * dpr)) {
		canvas.width = Math.round(cw * dpr);
		canvas.height = Math.round(ch * dpr);
	}
	const ctx = canvas.getContext('2d');
	if (!ctx) return;
	ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
	ctx.clearRect(0, 0, cw, ch);
	if (landmarks.length === 0 || !video.videoWidth) return;

	// Abbildung normierter Videokoordinaten auf das sichtbare Element (cover/contain).
	const vw = video.videoWidth;
	const vh = video.videoHeight;
	const scale = (opts.fit ?? 'cover') === 'cover' ? Math.max(cw / vw, ch / vh) : Math.min(cw / vw, ch / vh);
	const dw = vw * scale;
	const dh = vh * scale;
	const ox = (cw - dw) / 2;
	const oy = (ch - dh) / 2;
	const map = (p: Point) => ({ x: ox + p.x * dw, y: oy + p.y * dh });
	const visible = (p: Point) => (p.visibility ?? 1) >= MIN_VISIBILITY;

	ctx.lineCap = 'round';
	ctx.lineWidth = Math.max(3, Math.min(cw, ch) / 120);
	ctx.strokeStyle = opts.color;
	for (const [a, b] of SKELETON) {
		const pa = landmarks[a];
		const pb = landmarks[b];
		if (!pa || !pb || !visible(pa) || !visible(pb)) continue;
		const A = map(pa);
		const B = map(pb);
		ctx.beginPath();
		ctx.moveTo(A.x, A.y);
		ctx.lineTo(B.x, B.y);
		ctx.stroke();
	}
	ctx.fillStyle = opts.jointColor;
	const r = ctx.lineWidth * 1.3;
	for (let i = 11; i < Math.min(landmarks.length, 33); i++) {
		if (i >= 17 && i <= 22) continue; // Finger
		const p = landmarks[i];
		if (!visible(p)) continue;
		const P = map(p);
		ctx.beginPath();
		ctx.arc(P.x, P.y, r, 0, Math.PI * 2);
		ctx.fill();
	}
}

export function clearCanvas(canvas: HTMLCanvasElement) {
	canvas.getContext('2d')?.clearRect(0, 0, canvas.width, canvas.height);
}

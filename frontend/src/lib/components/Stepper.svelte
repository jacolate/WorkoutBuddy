<script lang="ts">
	// Zahleneingabe mit großen −/+ Buttons (gut für Touch).
	import { Minus, Plus } from '@lucide/svelte';
	import { formatClock } from '#lib/format.ts';

	let {
		value = $bindable(),
		min = 0,
		max = 999,
		step = 1,
		label,
		unit = '',
		time = false
	}: { value: number; min?: number; max?: number; step?: number; label?: string; unit?: string; time?: boolean } = $props();

	const clamp = (v: number) => Math.min(max, Math.max(min, v));
	let editing = $state(false);
	let draft = $state('');

	function commit() {
		editing = false;
		let v: number;
		if (time && draft.includes(':')) {
			const [m, s] = draft.split(':').map((x) => parseInt(x, 10) || 0);
			v = m * 60 + s;
		} else {
			v = parseInt(draft, 10);
		}
		if (!Number.isNaN(v)) value = clamp(v);
	}
</script>

<div>
	{#if label}<span class="label">{label}</span>{/if}
	<div class="flex h-12 items-stretch overflow-hidden rounded-xl border border-line bg-surface">
		<button type="button" class="grid w-12 shrink-0 place-items-center text-muted active:bg-surface-2 disabled:opacity-30" onclick={() => (value = clamp(value - step))} disabled={value <= min} aria-label="Weniger">
			<Minus class="size-5" />
		</button>
		{#if editing}
			<!-- svelte-ignore a11y_autofocus -->
			<input
				class="tabular w-full min-w-0 bg-transparent text-center text-lg font-semibold outline-none"
				inputmode={time ? 'text' : 'numeric'}
				bind:value={draft}
				onblur={commit}
				onkeydown={(e) => e.key === 'Enter' && (e.currentTarget as HTMLInputElement).blur()}
				autofocus
			/>
		{:else}
			<button
				type="button"
				class="tabular flex-1 text-center text-lg font-semibold"
				onclick={() => {
					draft = time ? formatClock(value) : String(value);
					editing = true;
				}}
			>
				{time ? formatClock(value) : value}{#if unit}<span class="ml-1 text-sm font-medium text-muted">{unit}</span>{/if}
			</button>
		{/if}
		<button type="button" class="grid w-12 shrink-0 place-items-center text-muted active:bg-surface-2 disabled:opacity-30" onclick={() => (value = clamp(value + step))} disabled={value >= max} aria-label="Mehr">
			<Plus class="size-5" />
		</button>
	</div>
</div>

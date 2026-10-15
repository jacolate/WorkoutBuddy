<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, ApiError } from '#lib/api.ts';
	import { app, theme } from '#lib/state.svelte.ts';
	import { uploads } from '#lib/camera/uploads.svelte.ts';
	import { House, ClockFading, Dumbbell, Settings, CloudUpload, CircleAlert } from '@lucide/svelte';

	let { children } = $props();

	let booting = $state(true);
	let bootError = $state<string | null>(null);

	const isPublic = $derived(page.url.pathname.startsWith('/login'));
	const fullscreen = $derived(page.url.pathname.startsWith('/play/'));

	const nav = [
		{ href: '/', label: 'Training', icon: House, match: (p: string) => p === '/' || p.startsWith('/workouts') },
		{ href: '/history', label: 'Verlauf', icon: ClockFading, match: (p: string) => p.startsWith('/history') || p.startsWith('/recordings') },
		{ href: '/exercises', label: 'Übungen', icon: Dumbbell, match: (p: string) => p.startsWith('/exercises') },
		{ href: '/settings', label: 'Einstellungen', icon: Settings, match: (p: string) => p.startsWith('/settings') }
	];

	async function boot() {
		booting = true;
		bootError = null;
		try {
			const me = await api.me();
			app.user = me.user;
			await app.load();
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				const next = page.url.pathname + page.url.search;
				await goto(`/login?next=${encodeURIComponent(next)}`, { replaceState: true });
			} else {
				bootError = (err as Error).message || 'Server nicht erreichbar';
			}
		} finally {
			booting = false;
		}
	}

	onMount(() => {
		theme.init();
		if (!isPublic) void boot();
		else booting = false;
	});

	// Nach dem Login (Navigation weg von /login) Daten laden.
	$effect(() => {
		if (!isPublic && !app.loaded && !booting && !bootError) void boot();
	});
</script>

{#if isPublic}
	{@render children()}
{:else if bootError}
	<div class="mx-auto grid min-h-dvh max-w-md place-items-center p-6 text-center">
		<div class="space-y-4">
			<CircleAlert class="mx-auto size-10 text-danger" />
			<p class="text-lg font-semibold">Verbindung fehlgeschlagen</p>
			<p class="text-muted">{bootError}</p>
			<button class="btn-primary w-full" onclick={boot}>Erneut versuchen</button>
		</div>
	</div>
{:else if booting || !app.loaded}
	<div class="grid min-h-dvh place-items-center">
		<div class="size-10 animate-spin rounded-full border-4 border-line border-t-accent"></div>
	</div>
{:else if fullscreen}
	{@render children()}
{:else}
	<div class="min-h-dvh md:flex">
		<!-- Desktop: Seitenleiste -->
		<aside class="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col border-r border-line bg-surface px-3 py-6 md:flex">
			<a href="/" class="mb-8 flex items-center gap-2 px-3 text-lg font-bold tracking-tight">
				<span class="grid size-8 place-items-center rounded-lg bg-accent text-accent-fg"><Dumbbell class="size-5" /></span>
				Workout
			</a>
			<nav class="flex flex-col gap-1">
				{#each nav as item (item.href)}
					{@const active = item.match(page.url.pathname)}
					<a
						href={item.href}
						class="flex min-h-11 items-center gap-3 rounded-xl px-3 font-medium transition {active
							? 'bg-accent-soft text-accent'
							: 'text-muted hover:bg-surface-2 hover:text-fg'}"
					>
						<item.icon class="size-5" />
						{item.label}
					</a>
				{/each}
			</nav>
		</aside>

		<main class="mx-auto w-full max-w-3xl px-4 pt-[max(1rem,env(safe-area-inset-top))] pb-[calc(5.5rem+env(safe-area-inset-bottom))] md:px-8 md:pt-8 md:pb-12">
			{@render children()}
		</main>

		<!-- Mobil: Tab-Leiste -->
		<nav
			class="fixed inset-x-0 bottom-0 z-30 border-t border-line bg-surface/90 pb-[env(safe-area-inset-bottom)] backdrop-blur-lg md:hidden"
		>
			<div class="mx-auto grid max-w-md grid-cols-4">
				{#each nav as item (item.href)}
					{@const active = item.match(page.url.pathname)}
					<a
						href={item.href}
						class="flex min-h-16 flex-col items-center justify-center gap-1 text-[11px] font-semibold transition {active ? 'text-accent' : 'text-muted'}"
					>
						<item.icon class="size-6" strokeWidth={active ? 2.4 : 2} />
						{item.label}
					</a>
				{/each}
			</div>
		</nav>
	</div>
{/if}

<!-- Upload-Status (überall sichtbar, auch im Player) -->
{#if uploads.pending > 0 || uploads.failed.length > 0}
	<div class="pointer-events-none fixed inset-x-0 top-[max(0.5rem,env(safe-area-inset-top))] z-50 flex justify-center px-4">
		<div class="pointer-events-auto flex items-center gap-2 rounded-full border border-line bg-surface px-4 py-2 text-sm font-medium shadow-card">
			{#if uploads.failed.length > 0}
				<CircleAlert class="size-4 text-danger" />
				<a href="/settings#uploads" class="text-danger">{uploads.failed.length} Upload(s) fehlgeschlagen</a>
			{:else}
				<CloudUpload class="size-4 animate-pulse text-accent" />
				Lade {uploads.pending} Aufnahme(n) hoch …
			{/if}
		</div>
	</div>
{/if}

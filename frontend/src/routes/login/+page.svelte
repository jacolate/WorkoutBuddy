<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '#lib/api.ts';
	import { app } from '#lib/state.svelte.ts';
	import { Dumbbell, LogIn } from '@lucide/svelte';

	let username = $state('');
	let password = $state('');
	let error = $state<string | null>(null);
	let busy = $state(false);

	function safeNext(): string {
		const next = page.url.searchParams.get('next') ?? '/';
		// Nur relative Pfade zulassen (kein Open Redirect).
		return next.startsWith('/') && !next.startsWith('//') && !next.startsWith('/login') ? next : '/';
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			const res = await api.login(username.trim(), password);
			app.reset();
			app.user = res.user;
			await goto(safeNext(), { replaceState: true });
		} catch (err) {
			error = (err as Error).message;
			password = '';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Anmelden · Workout</title></svelte:head>

<div class="grid min-h-dvh place-items-center px-4 py-10">
	<form class="w-full max-w-sm space-y-6" onsubmit={submit}>
		<div class="space-y-3 text-center">
			<div class="mx-auto grid size-14 place-items-center rounded-2xl bg-accent text-accent-fg shadow-card">
				<Dumbbell class="size-8" />
			</div>
			<h1 class="text-2xl font-bold tracking-tight">Workout</h1>
			<p class="text-muted">Bitte anmelden</p>
		</div>

		<div class="card space-y-4 p-5">
			<div>
				<label class="label" for="user">Benutzer</label>
				<input id="user" class="input" bind:value={username} autocomplete="username" autocapitalize="none" spellcheck="false" required />
			</div>
			<div>
				<label class="label" for="pw">Passwort</label>
				<input id="pw" class="input" type="password" bind:value={password} autocomplete="current-password" required />
			</div>
			{#if error}
				<p class="rounded-xl bg-danger-soft px-3 py-2 text-sm font-medium text-danger" role="alert">{error}</p>
			{/if}
			<button class="btn-primary w-full" type="submit" disabled={busy || !username || !password}>
				<LogIn class="size-5" />
				{busy ? 'Anmelden …' : 'Anmelden'}
			</button>
		</div>
		<p class="text-center text-xs text-muted">Du bleibst 30 Tage angemeldet.</p>
	</form>
</div>

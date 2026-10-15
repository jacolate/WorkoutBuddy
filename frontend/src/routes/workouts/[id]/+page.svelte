<script lang="ts">
	import { page } from '$app/state';
	import { api } from '#lib/api.ts';
	import WorkoutEditor from '#lib/components/WorkoutEditor.svelte';

	const id = $derived(Number(page.params.id));
	const workout = $derived(api.workout(id));
</script>

{#await workout}
	<div class="card h-40 animate-pulse"></div>
{:then w}
	{#key w.id}
		<WorkoutEditor workout={w} />
	{/key}
{:catch err}
	<div class="card p-6 text-center">
		<p class="mb-4 text-danger">{err.message}</p>
		<a href="/" class="btn-secondary">Zurück</a>
	</div>
{/await}

import { defineConfig } from 'vitest/config';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			// SPA: Go liefert index.html für alle unbekannten Pfade aus.
			adapter: adapter({
				pages: '../backend/web/dist',
				assets: '../backend/web/dist',
				fallback: 'index.html'
			})
		})
	],
	server: {
		proxy: {
			'/api': { target: 'http://localhost:8080', changeOrigin: false }
		}
	},
	test: {
		include: ['src/**/*.test.ts']
	}
});

import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import path from "path";

export default defineConfig({
	plugins: [svelte(), tailwindcss()],
	resolve: {
		alias: {
			$lib: path.resolve(__dirname, "./src/lib"),
		},
	},
	build: {
		outDir: "../dist",
		emptyOutDir: true,
	},
});

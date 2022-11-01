import { defineConfig } from "vite";
import path from "node:path";
import react from "@vitejs/plugin-react";
import dts from "vite-plugin-dts";
import tsConfigPaths from "vite-tsconfig-paths";

console.log(path.resolve(__dirname, "tsconfig.json").toString());
console.log(path.resolve(__dirname, "src/index.ts").toString());

export default defineConfig({
	plugins: [
		tsConfigPaths({
			projects: [path.resolve(__dirname, "tsconfig.json")],
		}),
		react(),
		dts({
			insertTypesEntry: true,
		}),
	],
	build: {
		sourcemap: true,
		lib: {
			entry: path.resolve(__dirname, "src/index.ts"),
			name: "pluto",
			formats: ["cjs", "es"],
			fileName: (format) => `pluto.${format}.js`,
		},
		rollupOptions: {
			external: ["react", "react-dom"],
			output: {
				globals: {
					react: "React",
					"react-dom": "ReactDOM",
				},
			},
		},
	},
	test: {
		globals: true,
		environment: "jsdom",
		setupFiles: ["setuptests.ts"],
	},
});

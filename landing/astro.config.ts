import react from "@astrojs/react";
import vercel from "@astrojs/vercel";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [react()],
  output: "server",
  adapter: vercel(),
  site: "https://synnaxlabs.com",
});

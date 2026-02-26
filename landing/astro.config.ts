import react from "@astrojs/react";
import vercel from "@astrojs/vercel";
import { grammar as arcGrammar } from "@synnaxlabs/arc";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [react()],
  output: "server",
  adapter: vercel(),
  markdown: {
    shikiConfig: {
      theme: "css-variables",
      langs: [arcGrammar],
    },
  },
  site: "https://synnaxlabs.com",
});

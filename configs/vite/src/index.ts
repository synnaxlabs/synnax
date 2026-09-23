// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "path";
import { type Plugin } from "vite";

import { injectCSS } from "./css.js";
import { declarations } from "./declarations.js";
import { checkExports, discoverModules, moduleEntries } from "./modules.js";

export { discoverModules, moduleExports } from "./modules.js";

export interface Options {
  name: string;
  /**
   * Whether the package includes its source maps in the npm tarball. Set false for a
   * package whose `files` field excludes them: production builds then write the maps
   * to disk without a sourceMappingURL comment, so the Console build can still chain
   * them and consumers are not pointed at a file the tarball omits.
   * @default true
   */
  publishSourcemaps?: boolean;
  /**
   * Build the package as a subpath-only module set: every `src/<name>/index.ts` is an
   * entry, `package.json` exports must match them, each source file stays its own
   * output file, and emitted modules keep their CSS imports.
   * @default false
   */
  modules?: boolean;
}

export const lib = ({
  name,
  publishSourcemaps = true,
  modules = false,
}: Options): Plugin[] => [
  {
    name: "vite-plugin-lib",
    config: (config) => {
      const prod = isProd();
      console.log(
        `\x1b[34m Synnax - ${prod ? "Production" : "Development"} mode\x1b[0m`,
      );
      const root = path.resolve(config.root ?? ".");
      const entry = modules
        ? moduleEntries(root, discoverModules(root))
        : path.join(root, "src/index.ts");
      return {
        resolve: { tsconfigPaths: true },
        build: {
          sourcemap: prod && !publishSourcemaps ? "hidden" : true,
          minify: prod,
          cssCodeSplit: modules,
          lib: {
            name,
            formats: ["es"],
            fileName: (_, entryName) => `${outputName(name, entryName)}.js`,
            entry,
            ...config.build?.lib,
          },
          rolldownOptions: modules
            ? { output: { preserveModules: true, preserveModulesRoot: "src" } }
            : {},
        },
      };
    },
    configResolved: (config) => {
      if (modules) checkExports(config.root, discoverModules(config.root));
    },
  },
  ...(modules ? [injectCSS()] : []),
  declarations(),
];

// Bundled dependencies keep their module paths under preserveModules. npm drops any
// `node_modules` directory from a tarball, so they land under `vendor/` instead.
const NODE_MODULES = "node_modules/";

const outputName = (name: string, entryName: string): string => {
  if (entryName === "index") return name;
  const at = entryName.lastIndexOf(NODE_MODULES);
  if (at === -1) return entryName;
  return `vendor/${entryName.slice(at + NODE_MODULES.length)}`;
};

export const isProd = () => process.env.SYNNAX_TS_ENV === "prod";

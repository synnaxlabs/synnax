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

import { declarations } from "./declarations.js";
import { checkExports, discoverModules } from "./modules.js";
import { ENTRY, transpile } from "./transpile.js";

export { type Layer, layers } from "./layers.js";
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
   * entry, `package.json` exports must match them, and each entry must only re-export.
   * Implies `transpiled`.
   * @default false
   */
  modules?: boolean;
  /**
   * Whether each source file is transpiled to its own output file instead of bundled,
   * so a consumer's bundler can drop the modules and namespace members it does not
   * use. The package must declare `sideEffects`.
   * @default false
   */
  transpiled?: boolean;
}

export const lib = ({
  name,
  publishSourcemaps = true,
  modules = false,
  transpiled = false,
}: Options): Plugin[] => {
  const prod = isProd();
  console.log(`\x1b[34m Synnax - ${prod ? "Production" : "Development"} mode\x1b[0m`);
  if (modules || transpiled)
    return [
      {
        name: "vite-plugin-lib",
        config: () => ({
          resolve: { tsconfigPaths: true },
          build: {
            write: false,
            lib: { name, formats: ["es"], entry: { [name]: ENTRY } },
          },
        }),
        configResolved: (config) => {
          if (modules) checkExports(config.root, discoverModules(config.root));
        },
      },
      transpile(modules),
    ];
  return [
    {
      name: "vite-plugin-lib",
      config: (config) => {
        const root = path.resolve(config.root ?? ".");
        return {
          resolve: { tsconfigPaths: true },
          build: {
            sourcemap: prod && !publishSourcemaps ? "hidden" : true,
            minify: prod,
            lib: {
              name,
              formats: ["es"],
              fileName: (_, entryName) =>
                `${entryName === "index" ? name : entryName}.js`,
              entry: path.join(root, "src/index.ts"),
              ...config.build?.lib,
            },
          },
        };
      },
    },
    declarations(),
  ];
};

export const isProd = () => process.env.SYNNAX_TS_ENV === "prod";

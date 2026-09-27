// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync, readdirSync, readFileSync } from "node:fs";
import path from "node:path";

/** A package export entry as written in `package.json`. */
export interface ExportEntry {
  types: string;
  default: string;
}

/** The module names of a package: every `src/<name>/index.ts`, sorted. */
export const discoverModules = (root: string): string[] =>
  readdirSync(path.join(root, "src"), { withFileTypes: true })
    .filter(
      (entry) =>
        entry.isDirectory() &&
        existsSync(path.join(root, "src", entry.name, "index.ts")),
    )
    .map((entry) => entry.name)
    .sort();

/** Vite `lib.entry` map for the given modules. */
export const moduleEntries = (
  root: string,
  modules: string[],
): Record<string, string> =>
  Object.fromEntries(
    modules.map((name) => [name, path.join(root, "src", name, "index.ts")]),
  );

/**
 * The `exports` map a subpath-only package must carry for the given modules, plus a
 * passthrough for static files under `dist/`.
 */
export const moduleExports = (
  modules: string[],
): Record<string, ExportEntry | string> => ({
  ...Object.fromEntries(
    modules.map((name) => [
      `./${name}`,
      { types: `./dist/src/${name}/index.d.ts`, default: `./dist/${name}.js` },
    ]),
  ),
  "./dist/*": "./dist/*",
});

/**
 * Throws when the package's `package.json` exports differ from the map its `src/`
 * layout implies. The message names the script that regenerates them.
 */
export const checkExports = (root: string, modules: string[]): void => {
  const manifest = JSON.parse(readFileSync(path.join(root, "package.json"), "utf8"));
  const actual = JSON.stringify(manifest.exports ?? {});
  const expected = JSON.stringify(moduleExports(modules));
  if (actual === expected) return;
  throw new Error(
    `package.json exports do not match the modules under src/. Run \`pnpm exports\` in ${root}.`,
  );
};

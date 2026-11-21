// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { execFileSync } from "node:child_process";
import { copyFileSync, mkdirSync, readFileSync, rmSync } from "node:fs";
import path from "node:path";

import type { Plugin } from "vite";

import { compileEmitted } from "./compiler.js";
import { files, fullySpecify, tscBin, TSCONFIG } from "./declarations.js";
import { discoverModules } from "./modules.js";

/**
 * The placeholder entry Vite bundles in place of the package's source. Library mode
 * resolves it against the root, so it arrives as an absolute path.
 */
export const ENTRY = "virtual:synnax-transpiled";
const RESOLVED_ENTRY = `\0${ENTRY}`;

const ENTRY_FIX =
  "Module entries must only re-export. Move each import into the module that uses " +
  "it, or page-wide CSS into the stylesheet an app imports once:";
const SIDE_EFFECT_IMPORT = /\bimport\s*["']([^"']+)["']/g;

const copyStylesheets = (srcDir: string, outDir: string): void => {
  for (const file of files(srcDir, (name) => name.endsWith(".css"))) {
    const dest = path.join(outDir, path.relative(srcDir, file));
    mkdirSync(path.dirname(dest), { recursive: true });
    copyFileSync(file, dest);
  }
};

// The package marks its JavaScript side-effect free, so a consumer's bundler skips an
// entry that only re-exports and drops any import it makes for side effects.
const checkEntries = (outDir: string, modules: string[]): void => {
  const offenses = modules.flatMap((name) => {
    const entry = path.join(outDir, name, "index.js");
    const code = readFileSync(entry, "utf8");
    return Array.from(
      code.matchAll(SIDE_EFFECT_IMPORT),
      ([, spec]) => `${name} imports ${spec}`,
    );
  });
  if (offenses.length === 0) return;
  throw new Error(`${ENTRY_FIX}\n${offenses.join("\n")}`);
};

/**
 * Builds one JavaScript file and declaration per source file, with fully specified
 * imports and stylesheets beside their modules. When `modules` is set, rejects module
 * entries that import for side effects. When `reactCompiled` is set, runs the React
 * Compiler over every emitted file.
 */
export const transpile = (modules: boolean, reactCompiled: boolean): Plugin => {
  let root = "";
  let outDir = "";
  let entries: string[] = [];
  return {
    name: "vite-plugin-transpile",
    apply: "build",
    configResolved: (config) => {
      root = config.root;
      outDir = path.resolve(root, config.build.outDir);
      if (modules) entries = discoverModules(root);
    },
    resolveId: (id) => (id.endsWith(ENTRY) ? RESOLVED_ENTRY : null),
    load: (id) => (id === RESOLVED_ENTRY ? "export {};" : null),
    buildStart() {
      rmSync(outDir, { recursive: true, force: true });
      const src = path.join(root, "src");
      this.addWatchFile(src);
      for (const file of files(src, () => true)) this.addWatchFile(file);
    },
    closeBundle: async () => {
      const start = performance.now();
      const args = ["-p", TSCONFIG, "--outDir", outDir];
      // The compiler finds components by their JSX, so tsc must leave it in place.
      if (reactCompiled) args.push("--jsx", "preserve");
      execFileSync(process.execPath, [tscBin(), ...args], {
        cwd: root,
        stdio: "inherit",
      });
      if (reactCompiled)
        await Promise.all(
          files(outDir, (name) => /\.jsx?$/.test(name)).map(compileEmitted),
        );
      fullySpecify(outDir, outDir, ".js");
      copyStylesheets(path.join(root, "src"), outDir);
      checkEntries(outDir, entries);
      console.log(`transpiled in ${Math.round(performance.now() - start)}ms`);
    },
  };
};

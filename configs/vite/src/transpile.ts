// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { execFileSync } from "node:child_process";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";

import { type Plugin } from "vite";

import { tscBin, TSCONFIG } from "./declarations.js";
import { discoverModules } from "./modules.js";

/**
 * The placeholder entry Vite bundles in place of the package's source. Library mode
 * resolves it against the root, so it arrives as an absolute path.
 */
export const ENTRY = "virtual:synnax-transpiled";
const RESOLVED_ENTRY = `\0${ENTRY}`;

const SPECIFIER = /((?:from|import)\s*\(?\s*)(["'])((?:@\/|\.{1,2}\/)[^"']*)\2/g;
const KEPT = /\.(css|js|json)$/;
const ENTRY_FIX =
  "Module entries must only re-export. Move each import into the module that uses " +
  "it, or page-wide CSS into the stylesheet an app imports once:";
const SIDE_EFFECT_IMPORT = /\bimport\s*["']([^"']+)["']/g;

const files = (dir: string, match: (name: string) => boolean): string[] =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return files(full, match);
    return match(entry.name) ? [full] : [];
  });

const relative = (from: string, to: string): string => {
  const rel = path.relative(from, to).split(path.sep).join("/");
  return rel.startsWith(".") ? rel : `./${rel}`;
};

// Node loads only fully specified relative imports, so each one must name its file.
const fullySpecify = (outDir: string): void => {
  const emitted = files(
    outDir,
    (name) => name.endsWith(".js") || name.endsWith(".d.ts"),
  );
  for (const file of emitted) {
    const dir = path.dirname(file);
    const source = readFileSync(file, "utf8");
    const rewritten = source.replace(
      SPECIFIER,
      (_, prefix: string, quote: string, spec: string) => {
        const target = spec.startsWith("@/")
          ? path.join(outDir, spec.slice(2))
          : path.resolve(dir, spec);
        let resolved = target;
        if (!KEPT.test(spec))
          if (existsSync(`${target}.js`)) resolved = `${target}.js`;
          else if (existsSync(path.join(target, "index.js")))
            resolved = path.join(target, "index.js");
          else throw new Error(`${file}: cannot resolve import "${spec}"`);
        return `${prefix}${quote}${relative(dir, resolved)}${quote}`;
      },
    );
    if (rewritten !== source) writeFileSync(file, rewritten);
  }
};

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
 * entries that import for side effects.
 */
export const transpile = (modules: boolean): Plugin => {
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
    closeBundle: () => {
      const start = performance.now();
      execFileSync(process.execPath, [tscBin(), "-p", TSCONFIG, "--outDir", outDir], {
        cwd: root,
        stdio: "inherit",
      });
      fullySpecify(outDir);
      copyStylesheets(path.join(root, "src"), outDir);
      checkEntries(outDir, entries);
      console.log(`transpiled in ${Math.round(performance.now() - start)}ms`);
    },
  };
};

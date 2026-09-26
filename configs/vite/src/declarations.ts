// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { execFileSync } from "node:child_process";
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

import { type Plugin } from "vite";

const TSCONFIG = "tsconfig.build.json";
const ALIAS = /(["'])@\/([^"']+)\1/g;

const tscBin = (): string => {
  const pkg = createRequire(import.meta.url).resolve("typescript/package.json");
  return path.join(path.dirname(pkg), "bin/tsc");
};

const declarationFiles = (dir: string): string[] =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return declarationFiles(full);
    return entry.name.endsWith(".d.ts") ? [full] : [];
  });

// tsc leaves `@/x` specifiers as written. Consumers resolve the emitted files without
// the package's tsconfig paths, so each one becomes a path relative to the file.
const rewriteAliases = (outDir: string): number => {
  const srcRoot = path.join(outDir, "src");
  let count = 0;
  for (const file of declarationFiles(outDir)) {
    const dir = path.dirname(file);
    const source = readFileSync(file, "utf8");
    const rewritten = source.replace(ALIAS, (_, quote: string, target: string) => {
      count++;
      const rel = path.relative(dir, path.join(srcRoot, target)).split(path.sep);
      const specifier = rel.join("/");
      return `${quote}${specifier.startsWith(".") ? specifier : `./${specifier}`}${quote}`;
    });
    if (rewritten !== source) writeFileSync(file, rewritten);
  }
  return count;
};

/**
 * Emits the package's type declarations into the build's `outDir` with the native
 * `tsc`, driven by the package's `tsconfig.build.json`, and rewrites `@/` imports in
 * the emitted files to relative paths. A type error fails the build.
 */
export const declarations = (): Plugin => {
  let root = "";
  let outDir = "";
  return {
    name: "vite-plugin-declarations",
    apply: "build",
    configResolved: (config) => {
      root = config.root;
      outDir = path.resolve(root, config.build.outDir);
    },
    closeBundle: () => {
      const start = performance.now();
      execFileSync(process.execPath, [tscBin(), "-p", TSCONFIG, "--outDir", outDir], {
        cwd: root,
        stdio: "inherit",
      });
      const count = rewriteAliases(outDir);
      const elapsed = Math.round(performance.now() - start);
      console.log(`declarations emitted in ${elapsed}ms (${count} aliases rewritten)`);
    },
  };
};

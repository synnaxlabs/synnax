// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { execFileSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

import type { Plugin } from "vite";

export const TSCONFIG = "tsconfig.build.json";
const SPECIFIER = /((?:from|import)\s*\(?\s*)(["'])(@\/[^"']*|\.{1,2}(?:\/[^"']*)?)\2/g;
const KEPT = /\.(css|js|json)$/;

export const tscBin = (): string => {
  const pkg = createRequire(import.meta.url).resolve("typescript/package.json");
  return path.join(path.dirname(pkg), "bin/tsc");
};

export const files = (dir: string, match: (name: string) => boolean): string[] =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return files(full, match);
    return match(entry.name) ? [full] : [];
  });

const relative = (from: string, to: string): string => {
  const rel = path.relative(from, to).split(path.sep).join("/");
  return rel.startsWith(".") ? rel : `./${rel}`;
};

/**
 * Rewrites each `@/` and relative import in the declaration files under `outDir`, and
 * in the JavaScript files when `ext` is `.js`, to a fully specified relative path, as
 * Node and `nodenext` type resolution require. `@/` resolves against `srcRoot`. `ext`
 * is the extension of the emitted files that prove a module exists. Throws on an
 * import that matches no emitted file.
 */
export const fullySpecify = (
  outDir: string,
  srcRoot: string,
  ext: ".js" | ".d.ts",
): void => {
  const emitted = files(outDir, (name) => name.endsWith(".d.ts") || name.endsWith(ext));
  for (const file of emitted) {
    const dir = path.dirname(file);
    const source = readFileSync(file, "utf8");
    const rewritten = source.replace(
      SPECIFIER,
      (_, prefix: string, quote: string, spec: string) => {
        const target = spec.startsWith("@/")
          ? path.join(srcRoot, spec.slice(2))
          : path.resolve(dir, spec);
        let resolved = target;
        if (!KEPT.test(spec))
          if (existsSync(`${target}${ext}`)) resolved = `${target}.js`;
          else if (existsSync(path.join(target, `index${ext}`)))
            resolved = path.join(target, "index.js");
          else throw new Error(`${file}: cannot resolve import "${spec}"`);
        return `${prefix}${quote}${relative(dir, resolved)}${quote}`;
      },
    );
    if (rewritten !== source) writeFileSync(file, rewritten);
  }
};

/**
 * Emits the package's type declarations into the build's `outDir` with the native
 * `tsc`, driven by the package's `tsconfig.build.json`, and fully specifies their
 * imports. A type error fails the build.
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
      fullySpecify(outDir, path.join(outDir, "src"), ".d.ts");
      console.log(`declarations emitted in ${Math.round(performance.now() - start)}ms`);
    },
  };
};

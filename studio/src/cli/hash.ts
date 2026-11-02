// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash } from "node:crypto";
import { existsSync, statSync } from "node:fs";
import { readFile } from "node:fs/promises";
import path from "node:path";

import { type Entry } from "@/manifest";

/**
 * Specifiers of the static import and export statements in a module. Prettier starts
 * each top-level statement at the first column, which keeps comments and strings out.
 */
const SPECIFIER =
  /^(?:(?:import|export)\s[^;"'`]*?\bfrom\s*|import\s*)["']([^"'\n]+)["']/gm;

const isFile = (file: string): boolean => existsSync(file) && statSync(file).isFile();

/** resolve returns the file a relative import specifier names. */
const resolve = (from: string, specifier: string): string => {
  const base = path.resolve(path.dirname(from), specifier);
  const file = [base, `${base}.ts`, `${base}.tsx`, path.join(base, "index.ts")].find(
    isFile,
  );
  if (file == null) throw new Error(`${from} imports "${specifier}", which is no file`);
  return file;
};

/**
 * modules returns `file` and every module under `root` that it imports, directly or
 * through another such module. `file` comes first, and the rest sort by path. Throws
 * when a relative import names no file.
 */
const modules = async (file: string, root: string): Promise<string[]> => {
  const found = new Set<string>();
  const visit = async (current: string): Promise<void> => {
    const source = await readFile(current, "utf8");
    for (const [, specifier] of source.matchAll(SPECIFIER)) {
      if (!specifier.startsWith(".")) continue;
      const dep = resolve(current, specifier);
      if (dep === file || found.has(dep)) continue;
      if (path.relative(root, dep).startsWith("..")) continue;
      found.add(dep);
      await visit(dep);
    }
  };
  await visit(file);
  return [file, ...[...found].sort()];
};

/**
 * entryHash hashes what a manifest entry's media depends on: its script, the modules
 * under `<root>/scripts` that the script imports, the entry's options, and the draft
 * flag. Modules under `<root>/src` are left out, so an engine change re-records nothing
 * until someone forces it.
 */
export const entryHash = async (
  entry: Entry,
  draft: boolean,
  root: string,
): Promise<string> => {
  const hash = createHash("sha256");
  const script = path.resolve(root, entry.script);
  for (const file of await modules(script, path.join(root, "scripts")))
    hash.update(await readFile(file));
  return hash.update(JSON.stringify({ entry, draft })).digest("hex");
};

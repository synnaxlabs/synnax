// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFile } from "node:fs/promises";
import path from "node:path";

import { filter, type Manifest } from "@/manifest";

/** Studio package root; every out/ path is resolved against it. */
export const ROOT = path.resolve(import.meta.dirname, "../..");
export const OUT_ROOT = path.join(ROOT, "out");
export const THEMES = ["light", "dark"] as const;

/** Stamp written beside a produced entry, recording what produced it. */
export interface Stamp {
  /** Hash of the script contents, entry options, and draft flag. */
  hash: string;
  draft: boolean;
  producedAt: string;
}

/** stampFile returns the path of an entry's production stamp. */
export const stampFile = (id: string): string =>
  path.join(OUT_ROOT, id, "produce.json");

/** readStamp returns an entry's production stamp, or null when it has none. */
export const readStamp = async (id: string): Promise<Stamp | null> => {
  try {
    return JSON.parse(await readFile(stampFile(id), "utf8")) as Stamp;
  } catch {
    return null;
  }
};

/**
 * select loads videos.ts and returns the entries matching the pattern, exiting
 * with an error when none do. `exact` matches the full id instead of a
 * substring.
 */
export const select = async (pattern?: string, exact = false): Promise<Manifest> => {
  const manifest = (await import(path.join(ROOT, "videos.ts"))) as {
    default: Manifest;
  };
  const entries = exact
    ? manifest.default.filter((e) => e.id === pattern)
    : filter(manifest.default, pattern);
  if (entries.length === 0) {
    console.error(`no manifest entries match "${pattern ?? ""}"`);
    process.exit(1);
  }
  return entries;
};

/** run executes a CLI entry point, reporting a failure on stderr. */
export const run = (main: () => Promise<void>): void => {
  main().catch((err: unknown) => {
    console.error(err);
    process.exit(1);
  });
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync } from "node:fs";
import { readdir, rm, stat } from "node:fs/promises";
import path from "node:path";
import { parseArgs } from "node:util";

import { filter, type Manifest, videoName } from "@/manifest";

const usage = `usage: pnpm clean [filter] [options]
Frees disk from produced artifacts for manifest entries whose id contains
[filter]. By default removes only capture frames (the heavy part, gigabytes per
theme), keeping timelines, rendered videos, and stamps; re-rendering after a
frames clean requires recapturing.
  --videos    also remove rendered videos and production stamps
  --all       remove everything under out/ for matching entries
              (with no filter: the entire out/ directory)
  --dry-run   list what would be removed and how much space it frees`;

const ROOT = path.resolve(import.meta.dirname, "../..");
const OUT_ROOT = path.join(ROOT, "out");
const THEMES = ["light", "dark"] as const;

const dirSize = async (target: string): Promise<number> => {
  const info = await stat(target).catch(() => null);
  if (info == null) return 0;
  if (!info.isDirectory()) return info.size;
  const entries = await readdir(target);
  let total = 0;
  for (const entry of entries) total += await dirSize(path.join(target, entry));
  return total;
};

const gb = (bytes: number): string => `${(bytes / 1024 ** 3).toFixed(2)} GB`;

const main = async (): Promise<void> => {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      videos: { type: "boolean", default: false },
      all: { type: "boolean", default: false },
      "dry-run": { type: "boolean", default: false },
      help: { type: "boolean", default: false },
    },
  });
  if (values.help) {
    console.log(usage);
    return;
  }
  const pattern = positionals[0];

  if (values.all && pattern == null) {
    const size = await dirSize(OUT_ROOT);
    if (values["dry-run"]) {
      console.log(`would remove ${path.relative(ROOT, OUT_ROOT)} (${gb(size)})`);
      return;
    }
    await rm(OUT_ROOT, { recursive: true, force: true });
    console.log(`removed out/ (${gb(size)} freed)`);
    return;
  }

  const manifest = (await import(path.join(ROOT, "videos.ts"))) as {
    default: Manifest;
  };
  const entries = filter(manifest.default, pattern);
  if (entries.length === 0) {
    console.error(`no manifest entries match "${pattern ?? ""}"`);
    process.exit(1);
  }

  const targets: string[] = [];
  for (const entry of entries) {
    const workDir = path.join(OUT_ROOT, entry.id);
    if (values.all) {
      targets.push(workDir);
      for (const theme of THEMES)
        targets.push(path.join(OUT_ROOT, videoName(entry.id, theme)));
      continue;
    }
    for (const theme of THEMES) targets.push(path.join(workDir, theme, "frames"));
    if (values.videos) {
      targets.push(path.join(workDir, "produce.json"));
      for (const theme of THEMES)
        targets.push(path.join(OUT_ROOT, videoName(entry.id, theme)));
    }
  }

  let freed = 0;
  for (const target of targets) {
    if (!existsSync(target)) continue;
    const size = await dirSize(target);
    freed += size;
    if (values["dry-run"]) {
      console.log(`would remove ${path.relative(ROOT, target)} (${gb(size)})`);
      continue;
    }
    await rm(target, { recursive: true, force: true });
    console.log(`removed ${path.relative(ROOT, target)} (${gb(size)})`);
  }
  console.log(values["dry-run"] ? `${gb(freed)} would be freed` : `${gb(freed)} freed`);
};

main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});

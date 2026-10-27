// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync } from "node:fs";
import { mkdir, readdir, readFile, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";

import {
  OUT_ROOT,
  readStamp,
  ROOT,
  run,
  select,
  type Stamp,
  stampFile,
  THEMES,
} from "@/cli/common";
import { entryHash } from "@/cli/hash";
import { runCapture, runRender, STUDIO_PORT } from "@/cli/pipeline";
import { type Entry, stillsDir, videoName } from "@/manifest";
import { type Timeline } from "@/timeline";

const usage = `usage: pnpm batch [filter] [options]
Produces every manifest entry (videos.ts) whose id contains [filter]: for each,
captures light and dark against fresh ephemeral cores and renders the themed
pair into the consolidated out/videos/ directory. A stills entry skips the render:
its stills land in out/stills/<id>/, and their regions at the entry's regions path.
Entries whose script, the modules it imports from scripts/, and options are
unchanged since their last successful production are skipped.
  --exact           treat [filter] as a full id instead of a substring
  --draft           fast review renders: higher crf + fast encoder preset
  --force           re-produce even when unchanged
  --keep-frames     keep captured frames after a successful render (default:
                    pruned, since each theme's frames run to gigabytes)
  --list            print matching entries and exit
  --url <url>       Console URL (default http://localhost:5173)
  --desktop-url <url>  Console URL of a Desktop build (VITE_DESKTOP=true), for
                    scripts that export desktop (default http://localhost:5175)
  --core-bin <path> synnax binary for the ephemeral cores
  --port <n>        core port for the ephemeral cores (default 9095)
  --jobs <n>        entries to produce concurrently (default 1); worker i gets
                    port + i, and its log prints when the entry finishes.
                    Entries pinning a port or marked serial produce first,
                    one at a time`;

const paths = (entry: Entry) => ({
  captureDir: (theme: "light" | "dark") => path.join(OUT_ROOT, entry.id, theme),
  video: (theme: "light" | "dark") => path.join(OUT_ROOT, videoName(entry.id, theme)),
  stills: path.join(OUT_ROOT, stillsDir(entry.id)),
});

interface ProduceEntryOptions {
  entry: Entry;
  url: string;
  desktopUrl?: string;
  draft: boolean;
  keepFrames: boolean;
  coreBin?: string;
  port?: number;
}

/** versionOf hashes named files, so their CDN keys change with their content. */
const versionOf = (files: Record<string, Buffer>): string => {
  const hash = createHash("sha256");
  for (const name of Object.keys(files).sort()) hash.update(name).update(files[name]);
  return hash.digest("hex").slice(0, 12);
};

const produceEntry = async ({
  entry,
  url,
  desktopUrl,
  draft,
  keepFrames,
  coreBin,
  port,
}: ProduceEntryOptions): Promise<void> => {
  const p = paths(entry);
  const capture = async (theme: "light" | "dark"): Promise<Timeline> => {
    const captureDir = p.captureDir(theme);
    await rm(path.join(captureDir, "stills"), { recursive: true, force: true });
    await mkdir(captureDir, { recursive: true });
    console.log(`  [${theme}] capturing...`);
    return await runCapture({
      scriptPath: path.resolve(ROOT, entry.script),
      outDir: captureDir,
      url,
      desktopUrl,
      theme,
      core: "ephemeral",
      coreBin,
      port: entry.port ?? port,
      hideCaret: entry.hideCaret ?? false,
      ...(entry.width != null && { width: entry.width }),
      ...(entry.height != null && { height: entry.height }),
      ...(entry.dsf != null && { dsf: entry.dsf }),
    });
  };
  if (entry.stills != null) {
    await rm(p.stills, { recursive: true, force: true });
    const stills: Record<string, Buffer> = {};
    const regions: string[] = [];
    for (const theme of THEMES) {
      await capture(theme);
      const dir = path.join(p.captureDir(theme), "stills");
      for (const file of await readdir(dir))
        if (file.endsWith(".webp"))
          stills[`${path.basename(file, ".webp")}-${theme}.webp`] = await readFile(
            path.join(dir, file),
          );
      regions.push(await readFile(path.join(dir, "regions.json"), "utf8"));
    }
    if (regions[0] !== regions[1])
      throw new Error("the light and dark stills have different regions");
    const version = versionOf(stills);
    await mkdir(path.join(p.stills, version), { recursive: true });
    for (const [file, data] of Object.entries(stills))
      await writeFile(path.join(p.stills, version, file), data);
    const regionsFile = path.resolve(ROOT, entry.stills.regions);
    await mkdir(path.dirname(regionsFile), { recursive: true });
    await writeFile(
      regionsFile,
      `${JSON.stringify({ version, ...JSON.parse(regions[0]) }, null, 2)}\n`,
    );
    console.log(`  wrote ${path.relative(ROOT, p.stills)} and ${entry.stills.regions}`);
  } else
    for (const theme of THEMES) {
      const captureDir = p.captureDir(theme);
      const timeline = await capture(theme);
      console.log(`  [${theme}] rendering ${timeline.meta.frames} frames...`);
      const outputLocation = p.video(theme);
      await mkdir(path.dirname(outputLocation), { recursive: true });
      await runRender({
        timeline,
        captureDir,
        outputLocation,
        target: entry.target,
        draft,
      });
      console.log(`  [${theme}] wrote ${path.relative(ROOT, outputLocation)}`);
      if (!keepFrames)
        await rm(path.join(captureDir, "frames"), { recursive: true, force: true });
    }
  const stamp: Stamp = {
    hash: await entryHash(entry, draft, ROOT),
    draft,
    producedAt: new Date().toISOString(),
  };
  await writeFile(stampFile(entry.id), JSON.stringify(stamp, null, 2));
};

interface Options {
  url: string;
  desktopUrl?: string;
  draft: boolean;
  keepFrames: boolean;
  coreBin?: string;
}

interface Result {
  ok: boolean;
  log: string;
}

/**
 * runChild produces one entry in a child process. Each capture points its
 * fixtures at its core through a process-wide environment variable, so workers
 * must not share a process.
 */
const runChild = async (id: string, port: number, opts: Options): Promise<Result> => {
  const args = [
    ...process.execArgv,
    fileURLToPath(import.meta.url),
    id,
    "--exact",
    "--force",
    "--url",
    opts.url,
    "--port",
    String(port),
  ];
  if (opts.draft) args.push("--draft");
  if (opts.keepFrames) args.push("--keep-frames");
  if (opts.coreBin != null) args.push("--core-bin", opts.coreBin);
  if (opts.desktopUrl != null) args.push("--desktop-url", opts.desktopUrl);
  return await new Promise<Result>((resolve) => {
    const proc = spawn(process.execPath, args, { stdio: ["ignore", "pipe", "pipe"] });
    let log = "";
    proc.stdout.on("data", (chunk: Buffer) => (log += chunk.toString()));
    proc.stderr.on("data", (chunk: Buffer) => (log += chunk.toString()));
    proc.on("exit", (code) => resolve({ ok: code === 0, log }));
  });
};

interface PoolResult {
  produced: number;
  failed: string[];
}

const runPool = async (
  entries: Entry[],
  jobs: number,
  basePort: number,
  opts: Options,
): Promise<PoolResult> => {
  const queue = [...entries];
  const failed: string[] = [];
  let produced = 0;
  const worker = async (index: number): Promise<void> => {
    while (queue.length > 0) {
      const entry = queue.shift();
      if (entry == null) return;
      console.log(`${entry.id}: started on port ${basePort + index}`);
      const { ok, log } = await runChild(entry.id, basePort + index, opts);
      process.stdout.write(log);
      if (ok) produced++;
      else failed.push(entry.id);
    }
  };
  await Promise.all(Array.from({ length: jobs }, async (_, i) => await worker(i)));
  return { produced, failed };
};

const main = async (): Promise<void> => {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      draft: { type: "boolean", default: false },
      force: { type: "boolean", default: false },
      "keep-frames": { type: "boolean", default: false },
      list: { type: "boolean", default: false },
      exact: { type: "boolean", default: false },
      url: { type: "string", default: "http://localhost:5173" },
      "desktop-url": { type: "string" },
      "core-bin": { type: "string" },
      port: { type: "string" },
      jobs: { type: "string" },
      help: { type: "boolean", default: false },
    },
  });
  if (values.help) {
    console.log(usage);
    return;
  }
  const entries = await select(positionals[0], values.exact);
  if (values.list) {
    for (const e of entries) console.log(`${e.id}  (${e.script})`);
    return;
  }

  const failed: string[] = [];
  let skipped = 0;
  let produced = 0;
  const stale: Entry[] = [];
  for (const entry of entries) {
    const p = paths(entry);
    const hash = await entryHash(entry, values.draft, ROOT);
    const stamp = await readStamp(entry.id);
    const present =
      entry.stills != null
        ? existsSync(p.stills) && existsSync(path.resolve(ROOT, entry.stills.regions))
        : THEMES.every((theme) => existsSync(p.video(theme)));
    const fresh = stamp?.hash === hash && present;
    if (fresh && !values.force) {
      console.log(`${entry.id}: unchanged, skipping`);
      skipped++;
      continue;
    }
    stale.push(entry);
  }

  const jobs = Math.max(1, Number(values.jobs ?? 1));
  const basePort = values.port != null ? Number(values.port) : STUDIO_PORT;
  // Pinned and serial entries bind fixed ports, so they cannot run beside each other.
  const pooled =
    jobs > 1 ? stale.filter((e) => e.port == null && e.serial !== true) : [];
  const serial = stale.filter((e) => !pooled.includes(e));
  for (const entry of serial) {
    console.log(`${entry.id}:`);
    try {
      await produceEntry({
        entry,
        url: values.url,
        desktopUrl: values["desktop-url"],
        draft: values.draft,
        keepFrames: values["keep-frames"],
        coreBin: values["core-bin"],
        ...(values.port != null && { port: Number(values.port) }),
      });
      produced++;
    } catch (err) {
      console.error(`${entry.id}: FAILED`, err);
      failed.push(entry.id);
    }
  }
  if (pooled.length > 0) {
    const result = await runPool(pooled, jobs, basePort, {
      url: values.url,
      desktopUrl: values["desktop-url"],
      draft: values.draft,
      keepFrames: values["keep-frames"],
      coreBin: values["core-bin"],
    });
    produced += result.produced;
    failed.push(...result.failed);
  }

  const failures = failed.length > 0 ? `:\n  ${failed.join("\n  ")}` : "";
  console.log(
    `\n${produced} produced, ${skipped} skipped, ${failed.length} failed${failures}`,
  );
  if (failed.length > 0) process.exit(1);
};

run(main);

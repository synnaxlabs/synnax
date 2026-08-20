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
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";

import { runCapture, runRender, STUDIO_PORT } from "@/cli/pipeline";
import { type Entry, filter, type Manifest, videoName } from "@/manifest";

const usage = `usage: pnpm batch [filter] [options]
Produces every manifest entry (videos.ts) whose id contains [filter]: for each,
captures light and dark against fresh ephemeral cores and renders the themed
pair into the consolidated out/videos/ directory. Entries whose script and
options are unchanged since their last successful production are skipped.
  --draft           fast review renders: 1080p cap + fast encoder preset
  --force           re-produce even when unchanged
  --keep-frames     keep captured frames after a successful render (default:
                    pruned, since each theme's frames run to gigabytes)
  --list            print matching entries and exit
  --url <url>       Console URL (default http://localhost:5173)
  --core-bin <path> synnax binary for the ephemeral cores
  --port <n>        core port for the ephemeral cores (default 9095)
  --jobs <n>        entries to produce concurrently (default 1); worker i gets
                    port + i, and its log prints when the entry finishes`;

const ROOT = path.resolve(import.meta.dirname, "../..");
const OUT_ROOT = path.join(ROOT, "out");

interface Stamp {
  /** Hash of the script contents, entry options, and draft flag. */
  hash: string;
  draft: boolean;
  producedAt: string;
}

const entryHash = async (entry: Entry, draft: boolean): Promise<string> => {
  const script = await readFile(path.resolve(ROOT, entry.script));
  return createHash("sha256")
    .update(script)
    .update(JSON.stringify({ entry, draft }))
    .digest("hex");
};

const paths = (entry: Entry) => {
  const workDir = path.join(OUT_ROOT, entry.id);
  return {
    workDir,
    stamp: path.join(workDir, "produce.json"),
    captureDir: (theme: "light" | "dark") => path.join(workDir, theme),
    video: (theme: "light" | "dark") => path.join(OUT_ROOT, videoName(entry.id, theme)),
  };
};

const readStamp = async (file: string): Promise<Stamp | null> => {
  try {
    return JSON.parse(await readFile(file, "utf8")) as Stamp;
  } catch {
    return null;
  }
};

interface ProduceEntryOptions {
  entry: Entry;
  url: string;
  draft: boolean;
  keepFrames: boolean;
  coreBin?: string;
  port?: number;
}

const THEMES = ["light", "dark"] as const;

const produceEntry = async ({
  entry,
  url,
  draft,
  keepFrames,
  coreBin,
  port,
}: ProduceEntryOptions): Promise<void> => {
  const p = paths(entry);
  for (const theme of THEMES) {
    const captureDir = p.captureDir(theme);
    await mkdir(captureDir, { recursive: true });
    console.log(`  [${theme}] capturing...`);
    const timeline = await runCapture({
      scriptPath: path.resolve(ROOT, entry.script),
      outDir: captureDir,
      url,
      theme,
      core: "ephemeral",
      coreBin,
      port,
      hideCaret: entry.hideCaret ?? false,
      ...(entry.width != null && { width: entry.width }),
      ...(entry.height != null && { height: entry.height }),
      ...(entry.dsf != null && { dsf: entry.dsf }),
    });
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
    hash: await entryHash(entry, draft),
    draft,
    producedAt: new Date().toISOString(),
  };
  await writeFile(p.stamp, JSON.stringify(stamp, null, 2));
};

interface Options {
  url: string;
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
  const manifest = (await import(path.join(ROOT, "videos.ts"))) as {
    default: Manifest;
  };
  const entries = values.exact
    ? manifest.default.filter((e) => e.id === positionals[0])
    : filter(manifest.default, positionals[0]);
  if (entries.length === 0) {
    console.error(`no manifest entries match "${positionals[0] ?? ""}"`);
    process.exit(1);
  }
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
    const hash = await entryHash(entry, values.draft);
    const stamp = await readStamp(p.stamp);
    const fresh =
      stamp?.hash === hash && THEMES.every((theme) => existsSync(p.video(theme)));
    if (fresh && !values.force) {
      console.log(`${entry.id}: unchanged, skipping`);
      skipped++;
      continue;
    }
    stale.push(entry);
  }

  const jobs = Math.max(1, Number(values.jobs ?? 1));
  const basePort = values.port != null ? Number(values.port) : STUDIO_PORT;
  if (jobs > 1) {
    const result = await runPool(stale, jobs, basePort, {
      url: values.url,
      draft: values.draft,
      keepFrames: values["keep-frames"],
      coreBin: values["core-bin"],
    });
    produced = result.produced;
    failed.push(...result.failed);
  } else
    for (const entry of stale) {
      console.log(`${entry.id}:`);
      try {
        await produceEntry({
          entry,
          url: values.url,
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

  const failures = failed.length > 0 ? `:\n  ${failed.join("\n  ")}` : "";
  console.log(
    `\n${produced} produced, ${skipped} skipped, ${failed.length} failed${failures}`,
  );
  if (failed.length > 0) process.exit(1);
};

main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});

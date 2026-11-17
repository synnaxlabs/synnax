// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync } from "node:fs";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { parseArgs } from "node:util";

import { z } from "zod";

import { bucket, CDN, OUT_ROOT, ROOT, run, select } from "@/cli/common";
import { versionOf } from "@/cli/hash";
import { type MailHero, runMailHeroes, runMailShot } from "@/cli/pipeline";
import { type Entry, stillKey } from "@/manifest";
import { type Region } from "@/remotion/Mail";

const usage = `usage: pnpm mail <id>[@seconds[:x,y,width]]... [options]
Makes the mail copies of the docs media that a release notes letter shows, as 2x JPEGs.
A hero still becomes a window on the docs hero gradient: give its id alone. A video
becomes a sharp shot of one frame: give the time of the frame, and optionally the region
to show, in pixels of the docs video (by default, the full width from the top). Shots
render from the light capture in out/<id>/light, so run
pnpm run batch <id> --force --keep-frames first. Uploads each copy to the docs CDN and
writes its URL to the portal's letter images. Requires DO_SPACES_KEY and
DO_SPACES_SECRET.
Options:
  --dry-run   write the copies to out/mail/ without uploading`;

const IMAGES = path.resolve(ROOT, "../site/portal/src/letters/images.json");

const stillZ = z.object({
  version: z.string(),
  stills: z.object({ window: z.unknown() }),
});

const ARG = /^([^@]+)(?:@(\d+(?:\.\d+)?)(?::(\d+),(\d+),(\d+))?)?$/;

interface Copy {
  id: string;
  seconds: number | null;
  region: Region | null;
}

const parseCopy = (arg: string): Copy => {
  const m = ARG.exec(arg);
  if (m == null) throw new Error(`${arg} is not <id>[@seconds[:x,y,width]]`);
  const [, id, seconds, x, y, width] = m;
  return {
    id,
    seconds: seconds == null ? null : Number(seconds),
    region: width == null ? null : { x: Number(x), y: Number(y), width: Number(width) },
  };
};

/** heroSource returns the CDN URL of the light render of a still. */
const heroSource = async ({ id, stills }: Entry): Promise<string> => {
  if (stills == null) throw new Error(`${id} is a video. Give the time of a frame`);
  const regions = await readFile(path.resolve(ROOT, stills.regions), "utf8");
  const { version } = stillZ.parse(JSON.parse(regions));
  return `${CDN}/${stillKey(id, `${version}/window-light.webp`)}`;
};

const outputOf = async (id: string): Promise<string> => {
  const output = path.join(OUT_ROOT, "mail", `${id}.jpg`);
  await mkdir(path.dirname(output), { recursive: true });
  return output;
};

const renderHeroes = async (entries: Entry[]): Promise<void> => {
  if (entries.length === 0) return;
  const dir = await mkdtemp(path.join(tmpdir(), "studio-mail-"));
  try {
    const heroes: MailHero[] = [];
    for (const [i, entry] of entries.entries()) {
      const url = await heroSource(entry);
      const res = await fetch(url);
      if (!res.ok) throw new Error(`${url} answered ${res.status}`);
      const source = `${i}${path.extname(url)}`;
      await writeFile(path.join(dir, source), Buffer.from(await res.arrayBuffer()));
      heroes.push({ source, output: await outputOf(entry.id) });
    }
    await runMailHeroes(dir, heroes);
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
};

const renderShot = async (
  entry: Entry,
  seconds: number,
  region: Region | null,
): Promise<void> => {
  if (entry.stills != null)
    throw new Error(`${entry.id} is a still. Give its id without a time`);
  const captureDir = path.join(OUT_ROOT, entry.id, "light");
  if (!existsSync(path.join(captureDir, "frames")))
    throw new Error(
      `${entry.id} has no capture. ` +
        `Run pnpm run batch ${entry.id} --force --keep-frames`,
    );
  await runMailShot({
    captureDir,
    target: entry.target,
    seconds,
    region,
    output: await outputOf(entry.id),
  });
};

const main = async (): Promise<void> => {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      "dry-run": { type: "boolean", default: false },
      help: { type: "boolean", default: false },
    },
  });
  if (values.help || positionals.length === 0) {
    console.log(usage);
    return;
  }
  const copies = new Map(positionals.map(parseCopy).map((c) => [c.id, c]));
  const entries = await select([...copies.keys()].join(","), true);
  const heroes: Entry[] = [];
  for (const entry of entries) {
    const { seconds, region } = copies.get(entry.id) as Copy;
    if (seconds == null) heroes.push(entry);
    else await renderShot(entry, seconds, region);
  }
  await renderHeroes(heroes);
  const outputs = await Promise.all(entries.map(async ({ id }) => await outputOf(id)));
  if (values["dry-run"]) {
    for (const output of outputs) console.log(path.relative(ROOT, output));
    return;
  }
  const put = await bucket();
  const images = z
    .record(z.string(), z.string())
    .parse(JSON.parse(await readFile(IMAGES, "utf8")));
  for (const [i, output] of outputs.entries()) {
    const { id } = entries[i];
    const version = versionOf({ "mail.jpg": await readFile(output) });
    const key = `docs/${id}/mail/${version}.jpg`;
    await put({ file: output, key, contentType: "image/jpeg" });
    images[id] = `${CDN}/${key}`;
    console.log(`${id} -> ${images[id]}`);
  }
  const sorted = Object.fromEntries(
    Object.entries(images).sort(([a], [b]) => a.localeCompare(b)),
  );
  await writeFile(IMAGES, `${JSON.stringify(sorted, null, 2)}\n`);
  console.log(`wrote ${path.relative(ROOT, IMAGES)}`);
};

run(main);

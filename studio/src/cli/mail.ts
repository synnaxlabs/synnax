// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { parseArgs } from "node:util";

import { z } from "zod";

import {
  bucket,
  CDN,
  OUT_ROOT,
  ROOT,
  run,
  select,
  versionsZ,
  VIDEO_VERSIONS,
} from "@/cli/common";
import { versionOf } from "@/cli/hash";
import { type MailCopy, runMailCopies } from "@/cli/pipeline";
import { type Entry, stillKey, videoKey } from "@/manifest";

const usage = `usage: pnpm mail <id>[@seconds]... [options]
Makes the mail copy of each docs video or hero still that a release notes letter shows:
its light render as a window on the docs hero gradient, as a 2x JPEG. A video needs
[seconds], the time of the frame to show. Uploads each copy to the docs CDN and writes
its URL to the portal's letter images. Reads the media from the CDN, so run pnpm upload
first. Requires DO_SPACES_KEY and DO_SPACES_SECRET.
Options:
  --dry-run   write the copies to out/mail/ without uploading`;

const IMAGES = path.resolve(ROOT, "../site/portal/src/letters/images.json");

const stillZ = z.object({
  version: z.string(),
  stills: z.object({ window: z.unknown() }),
});

/** sourceOf returns the CDN URL of the light render that an entry's mail copy shows. */
const sourceOf = async ({ id, stills, themed }: Entry): Promise<string> => {
  if (stills != null) {
    const regions = await readFile(path.resolve(ROOT, stills.regions), "utf8");
    const { version } = stillZ.parse(JSON.parse(regions));
    return `${CDN}/${stillKey(id, `${version}/window-light.webp`)}`;
  }
  const versions = versionsZ.parse(JSON.parse(await readFile(VIDEO_VERSIONS, "utf8")));
  const version = versions[id];
  if (version == null)
    throw new Error(`${id} is not on the CDN. Run pnpm upload first`);
  return `${CDN}/${videoKey(id, version, themed === false ? null : "light")}`;
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
  const times = new Map(
    positionals.map((arg) => {
      const [id, seconds] = arg.split("@");
      return [id, seconds == null ? null : z.coerce.number().parse(seconds)];
    }),
  );
  const entries = await select([...times.keys()].join(","), true);
  const dir = await mkdtemp(path.join(tmpdir(), "studio-mail-"));
  const copies: MailCopy[] = [];
  try {
    for (const [i, entry] of entries.entries()) {
      const url = await sourceOf(entry);
      const res = await fetch(url);
      if (!res.ok) throw new Error(`${url} answered ${res.status}`);
      const seconds = times.get(entry.id) ?? null;
      if ((entry.stills == null) !== (seconds != null))
        throw new Error(`${entry.id}: give a time for a video, and none for a still`);
      const source = `${i}${path.extname(url)}`;
      await writeFile(path.join(dir, source), Buffer.from(await res.arrayBuffer()));
      const output = path.join(OUT_ROOT, "mail", `${entry.id}.jpg`);
      await mkdir(path.dirname(output), { recursive: true });
      copies.push({ source, seconds, output });
    }
    await runMailCopies(dir, copies);
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
  if (values["dry-run"]) {
    for (const { output } of copies) console.log(path.relative(ROOT, output));
    return;
  }
  const put = await bucket();
  const images = z
    .record(z.string(), z.string())
    .parse(JSON.parse(await readFile(IMAGES, "utf8")));
  for (const [i, { output }] of copies.entries()) {
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

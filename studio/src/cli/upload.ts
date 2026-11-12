// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync } from "node:fs";
import { readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { parseArgs } from "node:util";

import {
  BUCKET,
  bucket,
  OUT_ROOT,
  type Put,
  readStamp,
  ROOT,
  run,
  select,
  THEMES,
  versionsZ,
  VIDEO_VERSIONS,
} from "@/cli/common";
import { versionOf } from "@/cli/hash";
import { stillKey, stillsDir, videoKey, videoName } from "@/manifest";

const usage = `usage: pnpm upload [filter] [options]
Uploads produced videos and stills for manifest entries whose id contains [filter]
to the docs CDN bucket (synnax/docs on DigitalOcean Spaces), where the docs site's
Video and Image components serve them, and writes each video's version to the docs
site. Requires credentials in the environment:
  DO_SPACES_KEY     Spaces access key id
  DO_SPACES_SECRET  Spaces secret key
Options:
  --exact         treat [filter] as a comma-separated list of full ids instead of a
                  substring
  --dry-run       print what would upload without uploading
  --allow-draft   permit uploading draft-quality renders (normally refused)`;

const main = async (): Promise<void> => {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      exact: { type: "boolean", default: false },
      "dry-run": { type: "boolean", default: false },
      "allow-draft": { type: "boolean", default: false },
      help: { type: "boolean", default: false },
    },
  });
  if (values.help) {
    console.log(usage);
    return;
  }
  const entries = await select(positionals[0], values.exact);

  const uploads: Put[] = [];
  const missing: string[] = [];
  const versions: Record<string, string> = {};
  for (const entry of entries) {
    if (!values["allow-draft"] && (await readStamp(entry.id))?.draft === true) {
      console.error(
        `${entry.id}: draft render; re-produce without --draft before uploading` +
          " (or pass --allow-draft)",
      );
      process.exit(1);
    }
    if (entry.stills != null) {
      const dir = path.join(OUT_ROOT, stillsDir(entry.id));
      if (!existsSync(dir)) {
        missing.push(stillsDir(entry.id));
        continue;
      }
      for (const file of await readdir(dir, { recursive: true }))
        if (file.endsWith(".webp"))
          uploads.push({
            file: path.join(dir, file),
            key: stillKey(entry.id, file),
            contentType: "image/webp",
          });
      continue;
    }
    // An unthemed page requests one file, so only the dark render ships.
    const themes = entry.themed === false ? (["dark"] as const) : THEMES;
    const absent = themes.filter(
      (theme) => !existsSync(path.join(OUT_ROOT, videoName(entry.id, theme))),
    );
    if (absent.length > 0) {
      missing.push(...absent.map((theme) => videoName(entry.id, theme)));
      continue;
    }
    const files: Record<string, Buffer> = {};
    for (const theme of themes)
      files[`${theme}.mp4`] = await readFile(
        path.join(OUT_ROOT, videoName(entry.id, theme)),
      );
    const version = versionOf(files);
    versions[entry.id] = version;
    for (const theme of themes)
      uploads.push({
        file: path.join(OUT_ROOT, videoName(entry.id, theme)),
        key: videoKey(entry.id, version, entry.themed === false ? null : theme),
        contentType: "video/mp4",
      });
  }
  if (missing.length > 0) {
    console.error(`missing media (run pnpm batch first):\n  ${missing.join("\n  ")}`);
    process.exit(1);
  }

  if (values["dry-run"]) {
    for (const u of uploads)
      console.log(`${path.relative(ROOT, u.file)} -> s3://${BUCKET}/${u.key}`);
    console.log(`${uploads.length} files (dry run, nothing uploaded)`);
    return;
  }

  const put = await bucket();
  for (const u of uploads) {
    process.stdout.write(`uploading ${u.key}... `);
    await put(u);
    console.log("done");
  }
  console.log(`${uploads.length} files uploaded`);
  if (Object.keys(versions).length === 0) return;
  const merged = {
    ...versionsZ.parse(JSON.parse(await readFile(VIDEO_VERSIONS, "utf8"))),
    ...versions,
  };
  const sorted = Object.fromEntries(
    Object.entries(merged).sort(([a], [b]) => a.localeCompare(b)),
  );
  await writeFile(VIDEO_VERSIONS, `${JSON.stringify(sorted, null, 2)}\n`);
  console.log(`wrote ${path.relative(ROOT, VIDEO_VERSIONS)}`);
};

run(main);

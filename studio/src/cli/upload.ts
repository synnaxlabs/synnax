// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync } from "node:fs";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { parseArgs } from "node:util";

import { cdnKey, filter, type Manifest, videoName } from "@/manifest";

const usage = `usage: pnpm upload [filter] [options]
Uploads produced videos for manifest entries whose id contains [filter] to the
docs CDN bucket (synnax/docs on DigitalOcean Spaces), where the docs site's
Video component serves them. Requires credentials in the environment:
  DO_SPACES_KEY     Spaces access key id
  DO_SPACES_SECRET  Spaces secret key
Options:
  --dry-run       print what would upload without uploading
  --allow-draft   permit uploading draft-quality renders (normally refused)`;

const ROOT = path.resolve(import.meta.dirname, "../..");
const OUT_ROOT = path.join(ROOT, "out");
const ENDPOINT = "https://nyc3.digitaloceanspaces.com";
const BUCKET = "synnax";
const THEMES = ["light", "dark"] as const;

interface Upload {
  file: string;
  key: string;
}

const main = async (): Promise<void> => {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      "dry-run": { type: "boolean", default: false },
      "allow-draft": { type: "boolean", default: false },
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
  const entries = filter(manifest.default, positionals[0]);
  if (entries.length === 0) {
    console.error(`no manifest entries match "${positionals[0] ?? ""}"`);
    process.exit(1);
  }

  const uploads: Upload[] = [];
  const missing: string[] = [];
  for (const entry of entries) {
    const stampFile = path.join(OUT_ROOT, entry.id, "produce.json");
    if (!values["allow-draft"] && existsSync(stampFile)) {
      const stamp = JSON.parse(await readFile(stampFile, "utf8")) as {
        draft?: boolean;
      };
      if (stamp.draft === true) {
        console.error(
          `${entry.id}: draft render; re-produce without --draft before uploading` +
            " (or pass --allow-draft)",
        );
        process.exit(1);
      }
    }
    for (const theme of THEMES) {
      const file = path.join(OUT_ROOT, videoName(entry.id, theme));
      if (existsSync(file)) uploads.push({ file, key: cdnKey(entry.id, theme) });
      else missing.push(videoName(entry.id, theme));
    }
  }
  if (missing.length > 0) {
    console.error(`missing videos (run pnpm batch first):\n  ${missing.join("\n  ")}`);
    process.exit(1);
  }

  if (values["dry-run"]) {
    for (const u of uploads)
      console.log(`${path.relative(ROOT, u.file)} -> s3://${BUCKET}/${u.key}`);
    console.log(`${uploads.length} files (dry run, nothing uploaded)`);
    return;
  }

  const key = process.env.DO_SPACES_KEY;
  const secret = process.env.DO_SPACES_SECRET;
  if (key == null || secret == null) {
    console.error("DO_SPACES_KEY and DO_SPACES_SECRET must be set");
    process.exit(1);
  }
  const { PutObjectCommand, S3Client } = await import("@aws-sdk/client-s3");
  const client = new S3Client({
    endpoint: ENDPOINT,
    region: "us-east-1",
    credentials: { accessKeyId: key, secretAccessKey: secret },
  });
  for (const u of uploads) {
    process.stdout.write(`uploading ${u.key}... `);
    await client.send(
      new PutObjectCommand({
        Bucket: BUCKET,
        Key: u.key,
        Body: await readFile(u.file),
        ACL: "public-read",
        ContentType: "video/mp4",
        CacheControl: "public, max-age=86400",
      }),
    );
    console.log("done");
  }
  console.log(`${uploads.length} files uploaded`);
};

main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});

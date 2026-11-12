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

import { z } from "zod";

import { filter, type Manifest } from "@/manifest";

/** Studio package root; every out/ path is resolved against it. */
export const ROOT = path.resolve(import.meta.dirname, "../..");
export const OUT_ROOT = path.join(ROOT, "out");
export const THEMES = ["light", "dark"] as const;

/** VIDEO_VERSIONS is the docs site's file of the version of each uploaded video. */
export const VIDEO_VERSIONS = path.resolve(
  ROOT,
  "../site/docs/src/components/media/videos.json",
);
export const versionsZ = z.record(z.string(), z.string());

/** Stamp written beside a produced entry, recording what produced it. */
export interface Stamp {
  /** Hash of the script, its modules under scripts/, entry options, and draft flag. */
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
 * select loads videos.ts and returns the entries matching the pattern, exiting with an
 * error when none do. `exact` reads the pattern as a comma-separated list of full ids
 * instead of a substring, and exits with an error when any id matches no entry.
 */
export const select = async (pattern?: string, exact = false): Promise<Manifest> => {
  const manifest = (await import(path.join(ROOT, "videos.ts"))) as {
    default: Manifest;
  };
  if (exact) {
    const ids = (pattern ?? "").split(",").map((id) => id.trim());
    const unknown = ids.filter((id) => !manifest.default.some((e) => e.id === id));
    if (unknown.length > 0) {
      console.error(`no manifest entries match "${unknown.join('", "')}"`);
      process.exit(1);
    }
    return manifest.default.filter((e) => ids.includes(e.id));
  }
  const entries = filter(manifest.default, pattern);
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

/** BUCKET is the DigitalOcean Spaces bucket of the docs CDN. */
export const BUCKET = "synnax";

/** CDN is the public URL of BUCKET. An object at key `k` is served at `${CDN}/${k}`. */
export const CDN = "https://synnax.nyc3.cdn.digitaloceanspaces.com";

export interface Put {
  file: string;
  key: string;
  contentType: string;
}

/**
 * bucket returns a function that puts files on the docs CDN. A key changes with its
 * content, so files are cached for good. Exits when DO_SPACES_KEY or DO_SPACES_SECRET
 * is not set.
 */
export const bucket = async (): Promise<(put: Put) => Promise<void>> => {
  const key = process.env.DO_SPACES_KEY;
  const secret = process.env.DO_SPACES_SECRET;
  if (key == null || secret == null) {
    console.error("DO_SPACES_KEY and DO_SPACES_SECRET must be set");
    process.exit(1);
  }
  const { PutObjectCommand, S3Client } = await import("@aws-sdk/client-s3");
  const client = new S3Client({
    endpoint: "https://nyc3.digitaloceanspaces.com",
    region: "us-east-1",
    credentials: { accessKeyId: key, secretAccessKey: secret },
  });
  return async ({ file, key, contentType }) => {
    await client.send(
      new PutObjectCommand({
        Bucket: BUCKET,
        Key: key,
        Body: await readFile(file),
        ACL: "public-read",
        ContentType: contentType,
        CacheControl: "public, max-age=31536000, immutable",
      }),
    );
  };
};

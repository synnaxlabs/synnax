// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/**
 * The manifest maps docs media ids to the scripts that produce them. For a video, the
 * id is the same string the docs site passes to its Video component: the CDN serves
 * `docs/<id>-light.mp4` and `docs/<id>-dark.mp4`. For stills, the CDN serves
 * `docs/<id>/<version>/<name>-<theme>.webp` for each still the script saves.
 */

export const entryZ = z.object({
  /** Docs media id, e.g. "console/line-plots/data-tab". */
  id: z
    .string()
    .regex(
      /^[a-z0-9_-]+(\/[a-z0-9_-]+)+$/,
      "ids are slash-separated lowercase segments",
    ),
  /** Script path, relative to the studio package root. */
  script: z.string().min(1),
  /** Capture viewport width in CSS px. */
  width: z.int().positive().optional(),
  /** Capture viewport height in CSS px. */
  height: z.int().positive().optional(),
  /** Capture device scale factor. */
  dsf: z.number().positive().optional(),
  /** Final render width target (e.g. "4k", "1080p", or pixels as a string). */
  target: z.string().optional(),
  /** Hides the text caret during capture. */
  hideCaret: z.boolean().optional(),
  /**
   * Mirrors the docs Video component's prop. A themed page asks the CDN for a
   * file per theme; an unthemed one asks for a single file, and the release
   * notes serve the dark render there. Both themes are still produced.
   */
  themed: z.boolean().optional(),
  /**
   * Pins the capture's core to this port, overriding the port the run assigns.
   * For shots that put the address on screen: the docs show 9090. Pinned
   * entries produce one at a time, since they cannot take a free port.
   */
  port: z.int().positive().optional(),
  /**
   * Produces the entry one at a time, never beside another. For shots that bind a
   * fixed port besides the core's, such as a simulator whose address the form shows.
   */
  serial: z.boolean().optional(),
  /**
   * Makes the entry a set of stills, which the script saves with `session.still`,
   * instead of a video. Batch writes the stills' regions to `regions`, a path relative
   * to the studio package root, for the docs page that draws them.
   */
  stills: z.object({ regions: z.string().min(1) }).optional(),
});
export type Entry = z.infer<typeof entryZ>;

export const manifestZ = entryZ.array().superRefine((entries, ctx) => {
  const seen = new Set<string>();
  for (const e of entries) {
    if (seen.has(e.id))
      ctx.addIssue({ code: "custom", message: `duplicate manifest id: ${e.id}` });
    seen.add(e.id);
  }
});
export type Manifest = z.infer<typeof manifestZ>;

/** define validates and returns a manifest; use it in videos.ts. */
export const define = (entries: Entry[]): Manifest => manifestZ.parse(entries);

/**
 * filter returns the entries whose id contains the pattern (substring match, so
 * "ranges" selects all of console/ranges/*). No pattern selects everything.
 */
export const filter = (manifest: Manifest, pattern?: string): Manifest =>
  pattern == null ? manifest : manifest.filter((e) => e.id.includes(pattern));

/**
 * videoName returns the output path of a rendered video relative to out/: all
 * videos land flat in one videos/ directory, with the id's slashes flattened.
 */
export const videoName = (id: string, theme: "light" | "dark"): string =>
  `videos/${id.replaceAll("/", "-")}-${theme}.mp4`;

/** stillsDir returns the directory, relative to out/, of an entry's stills. */
export const stillsDir = (id: string): string => `stills/${id}`;

/** stillKey returns the CDN object key of a file at `file` in an entry's stills. */
export const stillKey = (id: string, file: string): string => `docs/${id}/${file}`;

/**
 * cdnKey returns the object key the docs CDN serves the video under. An
 * unthemed entry takes the bare key, which is the one its page requests.
 */
export const cdnKey = (id: string, theme: "light" | "dark" | null): string =>
  theme == null ? `docs/${id}.mp4` : `docs/${id}-${theme}.mp4`;

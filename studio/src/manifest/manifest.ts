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
 * The manifest maps docs media ids to the video scripts that produce them. The
 * id is the same string the docs site passes to its Video component: the CDN
 * serves `docs/<id>-light.mp4` and `docs/<id>-dark.mp4`.
 */

export const entryZ = z.object({
  /** Docs media id, e.g. "console/line-plots/data-tab". */
  id: z
    .string()
    .regex(
      /^[a-z0-9-]+(\/[a-z0-9-]+)+$/,
      "ids are slash-separated kebab-case segments",
    ),
  /** Video script path, relative to the studio package root. */
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

/** videoName returns the output file name for an entry and theme. */
export const videoName = (id: string, theme: "light" | "dark"): string =>
  `${id}-${theme}.mp4`;

/** cdnKey returns the object key the docs CDN serves the video under. */
export const cdnKey = (id: string, theme: "light" | "dark"): string =>
  `docs/${videoName(id, theme)}`;

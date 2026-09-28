// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { colorZ } from "@/color/color";

/** A colored range of values starting at a threshold. */
export const bandZ = z.object({
  /** Unique identifier for the band. */
  key: z.string(),
  /** Lowest value, in the value's units, that the band paints. */
  threshold: z.number(),
  /** Fill painted while the value is in the band. */
  color: colorZ,
  /** True when the fill blinks while the value is in the band. */
  flashing: z.boolean().default(false),
});
export interface Band extends z.infer<typeof bandZ> {}

/** Maps a value to a color through threshold bands. */
export const scaleZ = z.object({
  /**
   * Threshold bands. A band paints values at or above its threshold and below the
   * next higher threshold.
   */
  bands: bandZ.array().default(() => []),
  /** True when the fill interpolates between band colors. */
  smooth: z.boolean().default(false),
});
export interface Scale extends z.infer<typeof scaleZ> {}

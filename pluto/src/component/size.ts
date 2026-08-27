// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** Every size step, from smallest to largest. */
export const SIZES = ["tiny", "small", "medium", "large", "huge"] as const;
/** Schema for a {@link Size}. */
export const size = z.enum(SIZES);
/**
 * The shared size scale. Components map it onto their own height, padding, and text
 * level, so a row of components at one size lines up.
 */
export type Size = z.infer<typeof size>;

/** Height in pixels for each {@link Size}. Mirrors the `--pluto-height-*` CSS vars. */
export const HEIGHTS: Record<Size, number> = {
  tiny: 21,
  small: 24,
  medium: 28,
  large: 36,
  huge: 48,
};

/** @returns true if the value is one of {@link SIZES}. */
export const isSize = (value: unknown): value is Size => size.safeParse(value).success;

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { text } from "@synnaxlabs/x";
import { z } from "zod";

import { type Component } from "@/component";
import { type theming } from "@/theming/aether";

// Typography levels are owned by the @synnaxlabs/x text package so that the
// server-side oracle schema and other visualizations (line plots, tables,
// etc.) reference the same canonical enum. Re-exported here so existing
// pluto callers can keep importing from @/text without churn.
export const LEVELS = text.LEVELS;
export const levelZ = text.levelZ;

/* Level of typography i.e paragraph and heading */
export type Level = text.Level;

export const downLevel = (level: Level): Level => {
  const index = LEVELS.indexOf(level);
  if (index === -1 || index === LEVELS.length - 1) return level;
  return LEVELS[index + 1];
};

export type Shade = theming.Shade;

export const weightZ = z.union([
  z.number(),
  z.enum(["normal", "bold", "bolder", "lighter"]),
]);

/* Weight sets the weight of the text */
export type Weight = z.infer<typeof weightZ>;

export const specZ = z.object({
  size: z.number(),
  weight: z.union([z.number(), z.string()]),
  lineHeight: z.number(),
  textTransform: z.string().optional(),
});

/* Defines a particular typography style */
export type Spec = z.infer<typeof specZ>;

/* Standardizes the typography levels for components of different sizes */
export const COMPONENT_SIZE_LEVELS: Record<Component.Size, Level> = {
  tiny: "small",
  small: "small",
  medium: "p",
  large: "h5",
  huge: "h2",
};

export const LEVEL_COMPONENT_SIZES: Record<Level, Component.Size> = {
  h1: "huge",
  h2: "huge",
  h3: "huge",
  h4: "large",
  h5: "small",
  p: "medium",
  small: "small",
};

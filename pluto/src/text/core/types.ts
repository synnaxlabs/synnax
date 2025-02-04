// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type theming } from "@/theming/aether";
import { type ComponentSize } from "@/util/component";

export const LEVELS = ["h1", "h2", "h3", "h4", "h5", "p", "small"] as const;
export const levelZ = z.enum(LEVELS);

const DOWN_LEVELS: Record<Level, Level> = {
  h1: "h2",
  h2: "h3",
  h3: "h4",
  h4: "h5",
  h5: "p",
  p: "small",
  small: "small",
};

export const downLevel = (level: Level): Level => DOWN_LEVELS[level];

const UP_LEVELS: Record<Level, Level> = {
  h1: "h1",
  h2: "h1",
  h3: "h2",
  h4: "h3",
  h5: "h4",
  p: "h5",
  small: "p",
};

export const upLevel = (level: Level): Level => UP_LEVELS[level];

/* Level of typography i.e paragraph and heading */
export type Level = z.infer<typeof levelZ>;

export type Shade = theming.Shade;

const STRING_WEIGHTS = ["normal", "bold", "bolder", "lighter"] as const;

export const weightZ = z.union([z.number(), z.enum(STRING_WEIGHTS)]);

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
export const ComponentSizeLevels: Record<ComponentSize, Level> = {
  small: "small",
  medium: "p",
  large: "h5",
  huge: "h2",
};

export const LevelComponentSizes: Record<Level, ComponentSize> = {
  h1: "huge",
  h2: "huge",
  h3: "huge",
  h4: "large",
  h5: "small",
  p: "medium",
  small: "small",
};

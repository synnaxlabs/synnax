// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type Level, LEVELS } from "@/text/types.gen";

/** Steps a typography level down by one rank; bottoms out at the smallest. */
export const downLevel = (level: Level): Level => {
  const index = LEVELS.indexOf(level);
  if (index === -1 || index === LEVELS.length - 1) return level;
  return LEVELS[index + 1];
};

export const weightZ = z.union([
  z.number(),
  z.enum(["normal", "bold", "bolder", "lighter"]),
]);

/** Weight sets the weight of the text. */
export type Weight = z.infer<typeof weightZ>;

export const specZ = z.object({
  size: z.number(),
  weight: weightZ,
  lineHeight: z.number(),
  textTransform: z.string().optional(),
});

/** Defines a particular typography style. */
export type Spec = z.infer<typeof specZ>;

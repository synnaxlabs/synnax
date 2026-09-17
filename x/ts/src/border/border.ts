// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { xy } from "@/spatial/xy";

/** Zod schema for {@link Radius}. */
export const radiusZ = z.object({
  topLeft: xy.xyZ,
  topRight: xy.xyZ,
  bottomLeft: xy.xyZ,
  bottomRight: xy.xyZ,
});

/**
 * Per-corner elliptical radii, each corner carrying an independent x and y component.
 * The caller sets the units, whether pixels or percentages of the element.
 */
export interface Radius extends z.infer<typeof radiusZ> {}

const numberCornersZ = z.object({
  topLeft: z.number(),
  topRight: z.number(),
  bottomLeft: z.number(),
  bottomRight: z.number(),
});

/** Zod schema for {@link CrudeRadius}. */
export const crudeRadiusZ = z.union([z.number(), xy.xyZ, numberCornersZ, radiusZ]);

/** A {@link Radius} in any of its shorthand forms. */
export type CrudeRadius = z.infer<typeof crudeRadiusZ>;

/**
 * @constructs Radius
 * @param radius - A scalar or a single {x, y} applied to every corner, or per-corner
 * numbers or {x, y} pairs.
 */
export const constructRadius = (radius: CrudeRadius): Radius => {
  if (typeof radius === "number" || "x" in radius) {
    const corner = xy.construct(radius);
    return {
      topLeft: corner,
      topRight: corner,
      bottomLeft: corner,
      bottomRight: corner,
    };
  }
  return {
    topLeft: xy.construct(radius.topLeft),
    topRight: xy.construct(radius.topRight),
    bottomLeft: xy.construct(radius.bottomLeft),
    bottomRight: xy.construct(radius.bottomRight),
  };
};

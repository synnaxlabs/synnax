// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { Color, crudeZ, ZERO } from "@/color/core/color";

export const stopZ = z.object({
  key: z.string(),
  color: crudeZ,
  position: z.number(),
  switched: z.boolean().optional(),
});

export type Stop = z.infer<typeof stopZ>;

export const gradientZ = z.array(stopZ);

export type Gradient = Stop[];

export const fromGradient = (gradient: Gradient, position: number): Color => {
  if (gradient.length === 0) return ZERO;

  gradient = gradient.slice().sort((a, b) => a.position - b.position);
  if (position <= gradient[0].position) return new Color(gradient[0].color);
  if (position >= gradient[gradient.length - 1].position)
    return new Color(gradient[gradient.length - 1].color);

  // Find the two stops between which the position lies
  for (let i = 0; i < gradient.length - 1; i++) {
    const start = gradient[i];
    const end = gradient[i + 1];

    if (position >= start.position && position <= end.position) {
      if (position === start.position) return new Color(start.color);

      if (position === end.position) return new Color(end.color);

      // Interpolate
      const t = (position - start.position) / (end.position - start.position);

      // Convert colors to RGBA
      const startColor = new Color(start.color);
      const endColor = new Color(end.color);

      const r = Math.round(startColor.r + t * (endColor.r - startColor.r));
      const g = Math.round(startColor.g + t * (endColor.g - startColor.g));
      const b = Math.round(startColor.b + t * (endColor.b - startColor.b));
      const a = startColor.a + t * (endColor.a - startColor.a); // Interpolate alpha directly

      return new Color([r, g, b, a]);
    }
  }

  // If position didn't match any interval, return the last color
  return new Color(gradient[gradient.length - 1].color);
};

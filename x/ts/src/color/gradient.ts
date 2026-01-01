// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import {
  aValue,
  bValue,
  type Color,
  construct,
  crudeZ,
  gValue,
  rValue,
  ZERO,
} from "@/color/color";

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
  if (position <= gradient[0].position) return construct(gradient[0].color);
  if (position >= gradient[gradient.length - 1].position)
    return construct(gradient[gradient.length - 1].color);

  for (let i = 0; i < gradient.length - 1; i++) {
    const start = gradient[i];
    const end = gradient[i + 1];

    if (position < start.position || position > end.position) continue;
    if (position === start.position) return construct(start.color);
    if (position === end.position) return construct(end.color);
    const t = (position - start.position) / (end.position - start.position);
    const startColor = construct(start.color);
    const endColor = construct(end.color);

    const r = Math.round(
      rValue(startColor) + t * (rValue(endColor) - rValue(startColor)),
    );
    const g = Math.round(
      gValue(startColor) + t * (gValue(endColor) - gValue(startColor)),
    );
    const b = Math.round(
      bValue(startColor) + t * (bValue(endColor) - bValue(startColor)),
    );
    const a = aValue(startColor) + t * (aValue(endColor) - aValue(startColor));

    return construct([r, g, b, a]);
  }

  return construct(gradient[gradient.length - 1].color);
};

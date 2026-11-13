// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Color, construct, type Crude, TRANSPARENT } from "@/color/color";

/** A color at a position along a gradient. */
export interface Stop {
  position: number;
  color: Crude;
}

export const fromGradient = (gradient: Stop[], position: number): Color => {
  if (gradient.length === 0) return TRANSPARENT;

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

    const mix = (i: number): number =>
      Math.round(startColor[i] + t * (endColor[i] - startColor[i]));
    return [mix(0), mix(1), mix(2), mix(3)];
  }

  return construct(gradient[gradient.length - 1].color);
};

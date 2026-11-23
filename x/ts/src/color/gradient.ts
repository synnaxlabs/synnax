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
  const sorted = gradient.toSorted((a, b) => a.position - b.position);
  const first = sorted[0];
  const last = sorted.at(-1);
  if (first == null || last == null) return TRANSPARENT;
  if (position <= first.position) return construct(first.color);
  if (position >= last.position) return construct(last.color);

  let start = first;
  for (const end of sorted.slice(1)) {
    if (position < start.position || position > end.position) {
      start = end;
      continue;
    }
    if (position === start.position) return construct(start.color);
    if (position === end.position) return construct(end.color);
    const t = (position - start.position) / (end.position - start.position);
    const [r0, g0, b0, a0] = construct(start.color);
    const [r1, g1, b1, a1] = construct(end.color);
    const mix = (from: number, to: number): number =>
      Math.round(from + t * (to - from));
    return [mix(r0, r1), mix(g0, g1), mix(b0, b1), mix(a0, a1)];
  }

  return construct(last.color);
};

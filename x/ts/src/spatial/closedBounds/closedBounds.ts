// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@/math";
import type { numeric } from "@/numeric";
import { type ClosedBounds, closedBoundsZ } from "@/spatial/base";

export { type ClosedBounds, closedBoundsZ };

/**
 * @returns true if target lies in [lower, upper], ends included. A bounds target must
 * lie wholly inside. Bounds with a lower above their upper contain no values.
 */
export const contains = <T extends numeric.Value>(
  bounds: ClosedBounds<T>,
  target: T | ClosedBounds<T>,
): boolean => {
  if (typeof target === "number" || typeof target === "bigint")
    return target >= bounds.lower && target <= bounds.upper;
  return target.lower >= bounds.lower && target.upper <= bounds.upper;
};

/** @returns upper - lower, which is negative when lower is above upper. */
export const span = <T extends numeric.Value>(bounds: ClosedBounds<T>): T =>
  math.sub(bounds.upper, bounds.lower);

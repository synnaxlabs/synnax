// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Bounds, bounds, type CrudeBounds } from "@/spatial/base";

export { type Bounds, bounds };

export type Crude = CrudeBounds;

export const construct = (lower: number | Crude, upper?: number): Bounds => {
  const b = { lower: 0, upper: 0 };
  if (typeof lower === "number") {
    if (upper != null) {
      b.lower = lower;
      b.upper = upper;
    } else {
      b.lower = 0;
      b.upper = lower;
    }
  } else if (Array.isArray(lower)) {
    [b.lower, b.upper] = lower;
  } else {
    b.lower = lower.lower;
    b.upper = lower.upper;
  }
  return makeValid(b);
};

export const ZERO = { lower: 0, upper: 0 };

export const INFINITE = { lower: -Infinity, upper: Infinity };

export const DECIMAL = { lower: 0, upper: 1 };

export const CLIP = { lower: -1, upper: 1 };

export const equals = (a: Bounds, b: Bounds): boolean =>
  a.lower === b.lower && a.upper === b.upper;

export const makeValid = (a: Bounds): Bounds => {
  if (a.lower > a.upper) return { lower: a.upper, upper: a.lower };
  return a;
};

export const clamp = (bounds: Bounds, target: number): number => {
  if (target < bounds.lower) return bounds.lower;
  if (target >= bounds.upper) return bounds.upper - 1;
  return target;
};

export const contains = (bounds: Bounds, target: number): boolean =>
  target >= bounds.lower && target < bounds.upper;

export const overlapsWith = (a: Bounds, b: Bounds): boolean =>
  contains(a, a.lower) || contains(b, b.upper - 1);

export const span = (a: Bounds): number => a.upper - a.lower;

export const isZero = (a: Bounds): boolean => a.lower === 0 && a.upper === 0;

export const spanIsZero = (a: Bounds): boolean => span(a) === 0;

export const isFinite = (a: Bounds): boolean =>
  Number.isFinite(a.lower) && Number.isFinite(a.upper);

export const max = (bounds: Crude[]): Bounds => ({
  lower: Math.min(...bounds.map((b) => construct(b).lower)),
  upper: Math.max(...bounds.map((b) => construct(b).upper)),
});

export const min = (bounds: Crude[]): Bounds => ({
  lower: Math.max(...bounds.map((b) => construct(b).lower)),
  upper: Math.min(...bounds.map((b) => construct(b).upper)),
});

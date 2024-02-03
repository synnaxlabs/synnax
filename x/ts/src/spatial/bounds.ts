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

export const equals = (_a?: Bounds, _b?: Bounds): boolean => {
  if (_a == null && _b == null) return true;
  if (_a == null || _b == null) return false;
  const a = construct(_a);
  const b = construct(_b);
  return a?.lower === b?.lower && a?.upper === b?.upper;
}

export const makeValid = (a: Bounds): Bounds => {
  if (a.lower > a.upper) return { lower: a.upper, upper: a.lower };
  return a;
};

export const clamp = (bounds: Crude, target: number): number => {
  const _bounds = construct(bounds);
  if (target < _bounds.lower) return _bounds.lower;
  if (target >= _bounds.upper) return _bounds.upper - 1;
  return target;
};

export const contains = (bounds: Crude, target: number): boolean => {
  const _bounds = construct(bounds);
  return target >= _bounds.lower && target < _bounds.upper;
}

export const overlapsWith = (a: Crude, b: Crude): boolean => {
  const _a = construct(a);
  const _b = construct(b);
  return contains(_a, _a.lower) || contains(_b, _b.upper - 1);
}

export const span = (a: Crude): number => {
  const _a = construct(a);
  return _a.upper - _a.lower;
}

export const isZero = (a: Crude): boolean => {
  const _a = construct(a);
  return _a.lower === 0 && _a.upper === 0;
}

export const spanIsZero = (a: Crude): boolean => span(a) === 0;

export const isFinite = (a: Crude): boolean => {
  const _a = construct(a);
  return Number.isFinite(_a.lower) && Number.isFinite(_a.upper);
}

export const max = (bounds: Crude[]): Bounds => ({
  lower: Math.min(...bounds.map((b) => construct(b).lower)),
  upper: Math.max(...bounds.map((b) => construct(b).upper)),
});

export const min = (bounds: Crude[]): Bounds => ({
  lower: Math.max(...bounds.map((b) => construct(b).lower)),
  upper: Math.min(...bounds.map((b) => construct(b).upper)),
});

export const constructArray = (bounds: Crude): number[] => {
  const _bounds = construct(bounds);
  return Array.from({ length: span(bounds) }, (_, i) => i + _bounds.lower);
}
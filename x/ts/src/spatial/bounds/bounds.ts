// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@/math";
import { type numeric } from "@/numeric";
import { type Bounds, boundsZ, type NumberCouple } from "@/spatial/base";

export { type Bounds, boundsZ };

export type Crude<T extends numeric.Value = number> = Bounds<T> | NumberCouple<T>;

/**
 * A bounds object with either or both of lower and upper omitted. Only valid for
 * number bounds: `construct` fills a missing lower with -Infinity and a missing upper
 * with Infinity, neither of which has a bigint representation.
 */
export type PartialCrude = Partial<Bounds<number>>;

interface ConstructOptions {
  /** Swaps lower and upper when lower is greater. Defaults to true. */
  makeValid?: boolean;
}

/**
 * Constructs bounds from a bounds object, a `[lower, upper]` couple, a single value
 * (taken as the upper, with a lower of 0), or a lower and an upper. Number and bigint
 * bounds are both supported.
 */
export interface Construct {
  /**
   * Constructs a bounds object from a partial bounds object. A missing lower defaults
   * to -Infinity and a missing upper defaults to Infinity. Only valid for number
   * bounds; bigint partials are rejected because there is no bigint infinity.
   */
  (bounds: PartialCrude, options?: ConstructOptions): Bounds<number>;

  <T extends numeric.Value = number>(
    bounds: Crude<T>,
    options?: ConstructOptions,
  ): Bounds<T>;

  <T extends numeric.Value = number>(lower: T, upper?: T | ConstructOptions): Bounds<T>;

  <T extends numeric.Value = number>(
    lower: T | Crude<T>,
    upper?: T | ConstructOptions,
    options?: ConstructOptions,
  ): Bounds<T>;
}

export const construct: Construct = <T extends numeric.Value>(
  lower: T | Crude<T> | PartialCrude,
  upper?: T | ConstructOptions,
  options?: ConstructOptions,
): Bounds<T> => {
  const b: Bounds<T> = {} as const as Bounds<T>;
  if (typeof upper === "object") {
    options = upper;
    upper = undefined;
  }
  options = { makeValid: true, ...options };
  if (typeof lower === "number" || typeof lower === "bigint")
    if (upper != null) {
      b.lower = lower;
      b.upper = upper;
    } else {
      b.lower = math.zero(lower);
      b.upper = lower;
    }
  else if (Array.isArray(lower)) {
    if (lower.length !== 2) throw new Error("bounds: expected array of length 2");
    [b.lower, b.upper] = lower;
  } else {
    b.lower = (lower.lower ?? -Infinity) as T;
    b.upper = (lower.upper ?? Infinity) as T;
  }
  return options?.makeValid ? makeValid<T>(b) : b;
};

export const INFINITE: Bounds = Object.freeze({ lower: -Infinity, upper: Infinity });
/**
 * Bounds containing no values: the identity for {@link union} and the dual of
 * {@link INFINITE}. Use as the "no data" sentinel; {@link isFinite} rejects it.
 */
export const INVALID: Bounds = Object.freeze({ lower: Infinity, upper: -Infinity });
export const DECIMAL: Bounds = Object.freeze({ lower: 0, upper: 1 });

const asGiven = <T extends numeric.Value>(bounds: Crude<T>): Bounds<T> =>
  construct<T>(bounds, { makeValid: false });

export const equals = <T extends numeric.Value = number>(
  _a?: Crude<T>,
  _b?: Crude<T>,
): boolean => {
  if (_a == null && _b == null) return true;
  if (_a == null || _b == null) return false;
  const a = asGiven(_a);
  const b = asGiven(_b);
  return a.lower === b.lower && a.upper === b.upper;
};
/** @returns a, with lower and upper swapped when lower is greater. */
export const makeValid = <T extends numeric.Value = number>(
  a: Bounds<T>,
): Bounds<T> => {
  if (a.lower > a.upper) return { lower: a.upper, upper: a.lower };
  return a;
};

/**
 * Projects the target onto the closed interval [lower, upper]. Returns lower if the
 * target is below it, upper if the target is above it, and the target otherwise.
 *
 * The result may equal upper, which {@link contains} rejects. clamp answers "nearest
 * valid magnitude", not "which half-open bucket". To clamp an index into
 * [lower, upper), clamp against upper - 1.
 */
export const clamp = <T extends numeric.Value>(bounds: Crude<T>, target: T): T => {
  const _bounds = asGiven(bounds);
  if (target < _bounds.lower) return _bounds.lower;
  if (target > _bounds.upper) return _bounds.upper;
  return target;
};

/**
 * @returns true if target lies in the half-open interval [lower, upper). A bounds
 * target is instead tested closed on both ends. Bounds with a lower above their upper
 * contain no values.
 */
export const contains = <T extends numeric.Value>(
  bounds: Crude<T>,
  target: T | Crude<T>,
): boolean => {
  const _bounds = asGiven(bounds);
  if (typeof target === "number" || typeof target === "bigint")
    return target >= _bounds.lower && target < _bounds.upper;
  const _target = asGiven(target);
  return _target.lower >= _bounds.lower && _target.upper <= _bounds.upper;
};

/**
 * @returns true if a and b share at least one value. Bounds with a lower above their
 * upper overlap nothing.
 */
export const overlapsWith = <T extends numeric.Value>(
  a: Crude<T>,
  b: Crude<T>,
): boolean => {
  const _a = asGiven(a);
  const _b = asGiven(b);
  if (_a.lower > _a.upper || _b.lower > _b.upper) return false;
  if (_a.lower === _b.lower) return true;
  if (_b.upper === _a.lower || _b.lower === _a.upper) return false;
  return (
    contains<T>(_a, _b.upper) ||
    contains<T>(_a, _b.lower) ||
    contains<T>(_b, _a.upper) ||
    contains<T>(_b, _a.lower)
  );
};

/**
 * @returns the span of the given bounds i.e. upper - lower, which is negative when
 * lower is above upper.
 */
export const span = <T extends numeric.Value>(a: Crude<T>): T => {
  const _a = asGiven(a);
  return math.sub(_a.upper, _a.lower);
};

/**
 * @returns true if both the upper and lower bounds are not Infinity or -Infinity,
 * false otherwise.
 */
export const isFinite = (a: Crude): boolean => {
  const _a = asGiven(a);
  // By nature, bigints can only be finite.
  if (typeof _a.lower === "bigint") return true;
  return Number.isFinite(_a.lower) && Number.isFinite(_a.upper);
};

/** @returns the midpoint between the lower and upper bounds. */
export const mean = (a: Crude): number => {
  const _a = asGiven(a);
  return (_a.upper + _a.lower) / 2;
};

/**
 * @returns bounds that have the maximum span of the given bounds i.e. the min of all
 * of the lower bounds and the max of all of the upper bounds. Members are not
 * reordered, so {@link INVALID} entries act as the identity instead of widening the
 * result to infinity.
 */
export const union = (bounds: Crude[]): Bounds => {
  let lower = Infinity;
  let upper = -Infinity;
  for (const b of bounds) {
    const c = asGiven(b);
    lower = Math.min(lower, c.lower);
    upper = Math.max(upper, c.upper);
  }
  return { lower, upper };
};

/**
 * @returns bounds that have the minimum span of the given bounds i.e. the max of all
 * of the lower bounds and the min of all of the upper bounds. Note that this function
 * may create invalid bounds if the highest lower bound is greater than the lowest upper
 * bound. Members are not reordered.
 */
export const intersection = (bounds: Crude[]): Bounds => {
  let lower = -Infinity;
  let upper = Infinity;
  for (const b of bounds) {
    const c = asGiven(b);
    lower = Math.max(lower, c.lower);
    upper = Math.min(upper, c.upper);
  }
  return { lower, upper };
};

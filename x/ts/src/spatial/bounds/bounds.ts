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
      b.lower = (typeof lower === "bigint" ? 0n : 0) as T;
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

export const ZERO: Bounds = Object.freeze({ lower: 0, upper: 0 });
export const INFINITE: Bounds = Object.freeze({ lower: -Infinity, upper: Infinity });
/**
 * Bounds containing no values: the identity for {@link max} unions and the dual of
 * {@link INFINITE}. Use as the "no data" sentinel; {@link isFinite} rejects it.
 */
export const INVALID: Bounds = Object.freeze({ lower: Infinity, upper: -Infinity });
export const DECIMAL: Bounds = Object.freeze({ lower: 0, upper: 1 });
/** Clip space bounds i.e. a lower bound of -1 and an upper bound of 1. */
export const CLIP = Object.freeze({ lower: -1, upper: 1 });

export const equals = <T extends numeric.Value = number>(
  _a?: Crude<T>,
  _b?: Crude<T>,
): boolean => {
  if (_a == null && _b == null) return true;
  if (_a == null || _b == null) return false;
  const a = construct(_a);
  const b = construct(_b);
  return a?.lower === b?.lower && a?.upper === b?.upper;
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
  const _bounds = construct<T>(bounds);
  if (target < _bounds.lower) return _bounds.lower;
  if (target > _bounds.upper) return _bounds.upper;
  return target;
};

/**
 * @returns true if target lies in the half-open interval [lower, upper). A bounds
 * target is instead tested closed on both ends.
 */
export const contains = <T extends numeric.Value>(
  bounds: Crude<T>,
  target: T | Crude<T>,
): boolean => {
  const _bounds = construct(bounds);
  if (typeof target === "number" || typeof target === "bigint")
    return target >= _bounds.lower && target < _bounds.upper;
  const _target = construct(target);
  return _target.lower >= _bounds.lower && _target.upper <= _bounds.upper;
};

/** @returns true if a and b share at least one value. */
export const overlapsWith = <T extends numeric.Value>(
  a: Crude<T>,
  b: Crude<T>,
): boolean => {
  const _a = construct<T>(a);
  const _b = construct<T>(b);
  if (_a.lower === _b.lower) return true;
  if (_b.upper === _a.lower || _b.lower === _a.upper) return false;
  return (
    contains<T>(_a, _b.upper) ||
    contains<T>(_a, _b.lower) ||
    contains<T>(_b, _a.upper) ||
    contains<T>(_b, _a.lower)
  );
};

/** @returns the span of the given bounds i.e. upper - lower. */
export const span = <T extends numeric.Value>(a: Crude<T>): T => {
  const _a = construct<T>(a);
  return (_a.upper - _a.lower) as T;
};

/** @returns true if both the lower and upper bounds are 0, false otherwise. */
export const isZero = <T extends numeric.Value>(a: Crude<T>): boolean => {
  const _a = construct(a);
  if (typeof _a.lower === "bigint") return _a.lower === 0n && _a.upper === 0n;
  return _a.lower === 0 && _a.upper === 0;
};

/**
 * @returns true if the difference between the lower and upper bounds is 0,
 * false otherwise.
 */
export const spanIsZero = <T extends numeric.Value>(a: Crude<T>): boolean => {
  const sp = span<T>(a);
  return typeof sp === "number" ? sp === 0 : sp === 0n;
};

/**
 * @returns true if both the upper and lower bounds are not Infinity or -Infinity,
 * false otherwise.
 */
export const isFinite = (a: Crude): boolean => {
  const _a = construct(a);
  // By nature, bigints can only be finite.
  if (typeof _a.lower === "bigint") return true;
  return Number.isFinite(_a.lower) && Number.isFinite(_a.upper);
};

/** @returns the midpoint between the lower and upper bounds. */
export const mean = (a: Crude): number => {
  const _a = construct(a);
  return (_a.upper + _a.lower) / 2;
};

/**
 * @returns bounds that have the maximum span of the given bounds i.e. the min of all
 * of the lower bounds and the max of all of the upper bounds. Members are not
 * reordered, so {@link INVALID} entries act as the identity instead of widening the
 * result to infinity.
 */
export const max = (bounds: Crude[]): Bounds => ({
  lower: Math.min(...bounds.map((b) => construct(b, { makeValid: false }).lower)),
  upper: Math.max(...bounds.map((b) => construct(b, { makeValid: false }).upper)),
});

/**
 * @returns bounds that have the minimum span of the given bounds i.e. the max of all
 * of the lower bounds and the min of all of the upper bounds. Note that this function
 * may create invalid bounds if the highest lower bound is greater than the lowest upper
 * bound. Members are not reordered.
 */
export const min = (bounds: Crude[]): Bounds => ({
  lower: Math.max(...bounds.map((b) => construct(b, { makeValid: false }).lower)),
  upper: Math.min(...bounds.map((b) => construct(b, { makeValid: false }).upper)),
});

/**
 * @returns an array of integers from the lower bound to the upper bound of the given
 * bounds.
 */
export const linspace = <T extends numeric.Value = number>(bounds: Crude<T>): T[] => {
  const _bounds = construct(bounds);
  const isBigInt = typeof _bounds.lower === "bigint";
  return Array.from({ length: Number(span(bounds)) }, (_, i) => {
    if (isBigInt) return ((_bounds.lower as bigint) + BigInt(i)) as T;
    return (_bounds.lower as number) + i;
  }) as T[];
};

/**
 * Finds where target belongs in an ordered, non-overlapping array of bounds.
 *
 * @returns `index`, the bounds entry the target falls in, or where a new entry would
 * be inserted when it falls outside every one; and `position`, the offset of the
 * target inside that entry, or 0 when it falls outside.
 */
export const findInsertPosition = <T extends numeric.Value>(
  bounds: Array<Crude<T>>,
  target: T,
): { index: number; position: number } => {
  const _bounds = bounds.map((b) => construct<T>(b));
  const index = _bounds.findIndex((b) => contains<T>(b, target) || target < b.lower);
  if (index === -1) return { index: bounds.length, position: 0 };
  const b = _bounds[index];
  if (contains(b, target)) return { index, position: Number(target - b.lower) };
  return { index, position: 0 };
};

/** A plan for splicing a new bound into an ordered array of bounds. */
export interface InsertionPlan {
  /** How much to increase the lower bound of the new bound or decrease the upper bound
   * of the previous bound. */
  removeBefore: number;
  /** How much to decrease the upper bound of the new bound or increase the lower bound
   * of the next bound. */
  removeAfter: number;
  /** The index at which to insert the new bound. */
  insertInto: number;
  /** The number of bounds to remove from the array. */
  deleteInBetween: number;
}

const ZERO_PLAN: InsertionPlan = {
  removeBefore: 0,
  removeAfter: 0,
  insertInto: 0,
  deleteInBetween: 0,
};

/**
 * Builds a plan for splicing value into an ordered, non-overlapping array of bounds.
 *
 * @returns null when value already lies entirely inside an existing bound.
 */
export const buildInsertionPlan = <T extends numeric.Value>(
  bounds: Array<Crude<T>>,
  value: Crude<T>,
): InsertionPlan | null => {
  const _bounds = bounds.map((b) => construct<T>(b));
  const _target = construct(value);
  if (_bounds.length === 0) return ZERO_PLAN;
  const lower = findInsertPosition<T>(_bounds, _target.lower);
  const upper = findInsertPosition<T>(_bounds, _target.upper);
  // Target starts past every bound.
  if (lower.index === bounds.length) return { ...ZERO_PLAN, insertInto: bounds.length };
  // Target ends before every bound.
  if (upper.index === 0) return { ...ZERO_PLAN, removeAfter: upper.position };
  if (lower.index === upper.index) {
    if (lower.position !== 0 && upper.position !== 0) return null;
    return {
      removeAfter: upper.position,
      removeBefore: lower.position,
      insertInto: lower.index,
      deleteInBetween: 0,
    };
  }
  let deleteInBetween = upper.index - lower.index;
  let insertInto = lower.index;
  let removeBefore = math.sub(Number(span(_bounds[lower.index])), lower.position);
  if (lower.position !== 0) {
    deleteInBetween -= 1;
    insertInto += 1;
  } else removeBefore = 0;
  return {
    removeBefore,
    removeAfter: upper.position,
    insertInto,
    deleteInBetween,
  };
};

/**
 * Walks `dist` from `start` through an ordered, non-overlapping array of bounds,
 * skipping the values between them. A negative distance walks backwards. A start
 * outside every bound resumes from the nearest one in the direction of travel.
 * Running out of bounds stops at the last value in that direction.
 *
 * @example traverse([[0, 10], [20, 30]], 5, 10) // => 25
 */
export const traverse = <T extends numeric.Value = number>(
  bounds: Array<Crude<T>>,
  start: T,
  dist: T,
): T => {
  const _bounds = bounds.map((b) => construct(b));

  const dir = dist > 0 ? 1 : dist < 0 ? -1 : 0;

  if (dir === 0) return start;

  let remainingDist = dist;
  let currentPosition = start;

  while (math.equal(remainingDist, 0) === false) {
    const index = _bounds.findIndex((b) => {
      if (dir > 0) return currentPosition >= b.lower && currentPosition < b.upper;
      return currentPosition > b.lower && currentPosition <= b.upper;
    });

    if (index !== -1) {
      const b = _bounds[index];
      let distanceInBound: T;
      if (dir > 0) distanceInBound = math.sub(b.upper, currentPosition);
      else distanceInBound = math.sub(currentPosition, b.lower);

      if (distanceInBound > (0 as T)) {
        const moveDist = math.min(math.abs(remainingDist), distanceInBound);
        currentPosition = math.add(currentPosition, dir > 0 ? moveDist : -moveDist);
        remainingDist = math.sub<T>(remainingDist, dir > 0 ? moveDist : -moveDist);

        if (math.equal(remainingDist, 0)) return currentPosition;
        continue;
      }
    }

    if (dir > 0) {
      const nextBounds = _bounds.filter((b) => b.lower > currentPosition);
      if (nextBounds.length > 0) currentPosition = nextBounds[0].lower;
      else return currentPosition;
    } else {
      const prevBounds = _bounds.filter((b) => b.upper < currentPosition);
      if (prevBounds.length > 0)
        currentPosition = prevBounds[prevBounds.length - 1].upper;
      else return currentPosition;
    }
  }
  return currentPosition;
};

/**
 * @returns the count of values between a and b that fall inside an ordered,
 * non-overlapping array of bounds, skipping the values between them.
 *
 * @example distance([[0, 10], [20, 30]], 5, 25) // => 10
 */
export const distance = <T extends numeric.Value = number>(
  bounds: Array<Crude<T>>,
  a: T,
  b: T,
): T => {
  const _bounds = bounds.map((b) => construct<T>(b));

  if (a === b) return (typeof a === "bigint" ? 0n : 0) as T;

  const interval = a < b ? construct([a, b]) : construct([b, a]);

  let totalDistance: T = (typeof a === "bigint" ? 0n : 0) as T;

  for (const bound of _bounds) {
    const overlapLower = bound.lower > interval.lower ? bound.lower : interval.lower;
    const overlapUpper = bound.upper < interval.upper ? bound.upper : interval.upper;

    if (overlapLower < overlapUpper) {
      const overlapSpan = (overlapUpper - overlapLower) as T;
      // @ts-expect-error - typescript doesn't recognize that totalDistance is a number
      totalDistance = (totalDistance + overlapSpan) as T;
    }
  }

  return totalDistance;
};

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { abs, add, equal as mathEqual, min as mathMin, sub } from "@/math/math";
import { type numeric } from "@/numeric";
import { type Bounds, bounds, type CrudeBounds } from "@/spatial/base";

export { type Bounds, bounds };

export type Crude<T extends numeric.Value = number> = CrudeBounds<T>;

/** Options for the `construct` function. */
interface ConstructOptions {
  /**
   * If true (default), automatically swaps the lower and upper bounds if the lower bound
   * is greater than the upper bound. This ensures the resulting bounds are valid.
   *
   * @example
   * // With makeValid: true (default)
   * construct(10, 0) // => { lower: 0, upper: 10 }
   *
   * @example
   * // With makeValid: false
   * construct(10, 0, { makeValid: false }) // => { lower: 10, upper: 0 }
   */
  makeValid?: boolean;
}

export interface Construct {
  /**
   * Constructs a bounds object from various input formats. The function supports multiple
   * overloads to handle different input types:
   *
   * 1. From a crude bounds object or array:
   * ```typescript
   * construct({ lower: 0, upper: 10 }) // => { lower: 0, upper: 10 }
   * construct([0, 10]) // => { lower: 0, upper: 10 }
   * ```
   *
   * 2. From separate lower and upper values:
   * ```typescript
   * construct(0, 10) // => { lower: 0, upper: 10 }
   * construct(10) // => { lower: 0, upper: 10 }
   * ```
   *
   * The function supports both number and bigint types through the generic parameter T.
   * By default, T is number.
   *
   * Options:
   * - makeValid: If true (default), swaps lower and upper bounds if lower > upper
   *
   * @param bounds - The input bounds to construct from. Can be:
   *   - A bounds object with lower and upper properties
   *   - An array of length 2 [lower, upper]
   *   - A single number/bigint (treated as upper bound, with lower = 0)
   *   - Two numbers/bigints (lower and upper bounds)
   * @param options - Optional configuration for bounds construction
   * @returns A bounds object with lower and upper properties
   *
   * @example
   * // From bounds object
   * construct({ lower: 0, upper: 10 })
   * // => { lower: 0, upper: 10 }
   *
   * @example
   * // From array
   * construct([0, 10])
   * // => { lower: 0, upper: 10 }
   *
   * @example
   * // From separate values
   * construct(0, 10)
   * // => { lower: 0, upper: 10 }
   *
   * @example
   * // Single value (upper bound only)
   * construct(10)
   * // => { lower: 0, upper: 10 }
   *
   * @example
   * // With bigint
   * construct(0n, 10n)
   * // => { lower: 0n, upper: 10n }
   *
   * @example
   * // Invalid bounds (lower > upper)
   * construct(10, 0)
   * // => { lower: 0, upper: 10 } (bounds are swapped)
   */
  <T extends numeric.Value = number>(
    bounds: Crude<T>,
    options?: ConstructOptions,
  ): Bounds<T>;

  /**
   * Constructs a bounds object from separate lower and upper values.
   *
   * @param lower - The lower bound value
   * @param upper - The upper bound value. If omitted, lower is used as the upper bound
   * and 0 is used as the lower bound
   * @returns A bounds object with lower and upper properties
   */
  <T extends numeric.Value = number>(lower: T, upper?: T | ConstructOptions): Bounds<T>;

  <T extends numeric.Value = number>(
    lower: T | Crude,
    upper?: T | ConstructOptions,
    options?: ConstructOptions,
  ): Bounds<T>;
}

export const construct = <T extends numeric.Value>(
  lower: T | Crude<T>,
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
    b.lower = lower.lower;
    b.upper = lower.upper;
  }
  return options?.makeValid ? makeValid<T>(b) : b;
};

/** A lower and upper bound of 0. */
export const ZERO: Bounds = Object.freeze({ lower: 0, upper: 0 });
/** A lower bound of -Infinity and an upper bound of Infinity. */
export const INFINITE: Bounds = Object.freeze({ lower: -Infinity, upper: Infinity });
/** A lower bound of 0 and an upper bound of 1. */
export const DECIMAL: Bounds = Object.freeze({ lower: 0, upper: 1 });
/** Clip space bounds i.e. a lower bound of -1 and an upper bound of 1. */
export const CLIP = Object.freeze({ lower: -1, upper: 1 });

/**
 * Checks whether the given bounds are equal.
 *
 * @param _a - The first bounds to compare.
 * @param _b - The second bounds to compare.
 * @returns True if the bounds are equal, false otherwise.
 */
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
/**
 * Makes the given bounds valid by swapping the lower and upper bounds if the lower bound
 * is greater than the upper bound.
 * @param a  - The bounds to make valid.
 * @returns The valid bounds.
 */
export const makeValid = <T extends numeric.Value = number>(
  a: Bounds<T>,
): Bounds<T> => {
  if (a.lower > a.upper) return { lower: a.upper, upper: a.lower };
  return a;
};

/**
 * Clamps the given target value to the given bounds. If the target is less than the lower
 * bound, the lower bound is returned. If the target is greater than or equal to the upper
 * bound, the upper bound minus 1 is returned. Otherwise, the target is returned.
 *
 * @param bounds - The bounds to clamp the target to.
 * @param target - The target value to clamp.
 * @returns The clamped target value.
 */
export const clamp = <T extends numeric.Value>(bounds: Crude<T>, target: T): T => {
  const _bounds = construct<T>(bounds);
  if (target < _bounds.lower) return _bounds.lower;
  if (target >= _bounds.upper)
    return (_bounds.upper - ((typeof _bounds.upper === "number" ? 1 : 1n) as T)) as T;
  return target;
};

/**
 * Checks whether the given target value or bounds are within the given bounds.
 *
 * @param bounds - The bounds to check against.
 * @param target - The target value to check. Can either be a number or a bounds object.
 * @returns True if the target is within the bounds, false otherwise.
 */
export const contains = <T extends numeric.Value>(
  bounds: Crude<T>,
  target: T | CrudeBounds<T>,
): boolean => {
  const _bounds = construct(bounds);
  if (typeof target === "number" || typeof target === "bigint")
    return target >= _bounds.lower && target < _bounds.upper;
  const _target = construct(target);
  return _target.lower >= _bounds.lower && _target.upper <= _bounds.upper;
};

/**
 * Checks whether the given bounds overlap with each other.
 *
 * @param a - The first bounds to check.
 * @param b - The second bounds to check.
 * @returns True if the bounds overlap, false otherwise.
 */
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

/**
 * Returns the mean value between the lower and upper bounds.
 *
 * @param a - The bounds to find the mean of. Can be either a strict bounds object
 * with 'lower' and 'upper' properties or an array of length 2.
 * @returns The mean value between the lower and upper bounds.
 *
 * @example
 * bounds.mean([0, 10]) // => 5
 * bounds.mean({ lower: 0, upper: 10 }) // => 5
 */
export const mean = (a: Crude): number => {
  const _a = construct(a);
  return (_a.upper + _a.lower) / 2;
};

/**
 * @returns bounds that have the maximum span of the given bounds i.e. the min of all
 * of the lower bounds and the max of all of the upper bounds.
 */
export const max = (bounds: Crude[]): Bounds => ({
  lower: Math.min(...bounds.map((b) => construct(b).lower)),
  upper: Math.max(...bounds.map((b) => construct(b).upper)),
});

/**
 * @returns bounds that have the minimum span of the given bounds i.e. the max of all
 * of the lower bounds and the min of all of the upper bounds. Note that this function
 * may create invalid bounds if the highest lower bound is greater than the lowest upper
 * bound.
 */
export const min = (bounds: Crude[]): Bounds => ({
  lower: Math.max(...bounds.map((b) => construct(b).lower)),
  upper: Math.min(...bounds.map((b) => construct(b).upper)),
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
 * Finds the index and position where a target value should be inserted into an array
 * of bounds.
 *
 * Crucially, this function assumes that the bounds are ORDERED and NON-OVERLAPPING.
 *
 * @template T
 * @param {Array<Crude<T>>} bounds - An array of crude bounds. Each bound can either be
 * an array of length 2 or an object with `lower` and `upper` properties.
 * @param {T} target - The target value to insert.
 *
 * @returns {{ index: number, position: number }} An object containing:
 * - `index`: The index in the bounds array where the target belongs.
 * - `position`: The position within the bound where the target fits. If the target is
 * outside all bounds, the index will be where a new bound can be inserted.
 *
 * @example
 * // Target within an existing bound
 * const bounds = [[0, 10], [20, 30]];
 * const target = 5;
 * const result = findInsertPosition(bounds, target);
 * // { index: 0, position: 5 }
 *
 * @example
 * // Target greater than all bounds
 * const bounds = [[0, 10], [20, 30]];
 * const target = 35;
 * const result = findInsertPosition(bounds, target);
 *  // { index: 2, position: 0 }
 *
 * @example
 * // Target less than all bounds
 * const bounds = [[10, 20], [30, 40]];
 * const target = 5;
 * const result = findInsertPosition(bounds, target);
 *  // { index: 0, position: 0 }
 *
 * @example
 * // Target overlaps between bounds
 * const bounds = [[0, 10], [20, 30]];
 * const target = 15;
 * const result = findInsertPosition(bounds, target);
 *  // { index: 1, position: 0 }
 *
 * @example
 * // Empty bounds array
 * const bounds = [];
 * const target = 5;
 * const result = findInsertPosition(bounds, target);
 *  // { index: 0, position: 0 }
 *
 * @example
 * // Target exactly at lower bound
 * const bounds = [[0, 10], [20, 30]];
 * const target = 10;
 * const result = findInsertPosition(bounds, target);
 *  // { index: 1, position: 0 }
 *
 * @example
 * // Target exactly at upper bound
 * const bounds = [[0, 10], [20, 30]];
 * const target = 30;
 * const result = findInsertPosition(bounds, target);
 *  // { index: 2, position: 0 }
 *
 * @example
 * // Target inside bounds with exact fit
 * const bounds = [[0, 5], [5, 10]];
 * const target = 5;
 * const result = findInsertPosition(bounds, target);
 *  // { index: 1, position: 0 }
 *
 * @throws {Error} If invalid bounds are provided, such as bounds arrays not being of
 * length 2.
 *
 * See {@link construct} for constructing valid bounds.
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

/**
 * A plan for inserting a new bound into an ordered array of bounds.
 */
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
 * Build a plan for inserting a new bound into an ordered array of bounds. This function
 * is particularly useful for inserting a new array into a sorted array of array of arrays
 * that may overlap. The plan is used to determine how to splice the new array into the
 * existing array. The following are important constraints:
 *
 * Crucially, this function assumes that the bounds are ORDERED and NON-OVERLAPPING.
 *
 * 1. If the new bound is entirely contained within an existing bound, the new bound
 * is not inserted and the plan is null.
 *
 * @param bounds - An ordered array of bounds, where each bound is valid (i.e., lower <= upper)
 * and the lower bound of each bound is less than the upper bound of the next bound.
 * @param value - The new bound to insert.
 * @returns A plan for inserting the new bound into the array of bounds, or null if the
 * new bound is entirely contained within an existing bound. See the {@link InsertionPlan}
 * type for more details.
 */
export const buildInsertionPlan = <T extends numeric.Value>(
  bounds: Array<Crude<T>>,
  value: Crude<T>,
): InsertionPlan | null => {
  const _bounds = bounds.map((b) => construct<T>(b));
  const _target = construct(value);
  // No bounds to insert into, so just insert the new bound at the beginning of the array.
  if (_bounds.length === 0) return ZERO_PLAN;
  const lower = findInsertPosition<T>(_bounds, _target.lower);
  const upper = findInsertPosition<T>(_bounds, _target.upper);
  // Greater than all bounds,
  if (lower.index === bounds.length) return { ...ZERO_PLAN, insertInto: bounds.length };
  // Less than all bounds,
  if (upper.index === 0) return { ...ZERO_PLAN, removeAfter: upper.position };
  if (lower.index === upper.index) {
    // The case where the bound is entirely contained within an existing bound.
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
  let removeBefore = sub(Number(span(_bounds[lower.index])), lower.position);
  // If we're overlapping with the previous bound, we need to slice out one less
  // and insert one further up.
  if (lower.position !== 0) {
    deleteInBetween -= 1;
    insertInto += 1;
    // We're not overlapping with the previous bound, so don't need to remove anything
  } else removeBefore = 0;
  return {
    removeBefore,
    removeAfter: upper.position,
    insertInto,
    deleteInBetween,
  };
};

/**
 * Traverse the given bounds by the specified distance, starting from a given point, and
 * return the end point of the traversal. The traversal 'skips' over integers that are
 * not within the array of bounds, moving only within the defined bounds. Traversing
 * across multiple bounds is handled smoothly, with direction determined by the sign of
 * the distance.
 *
 * Crucially, this function assumes that the bounds are ORDERED and NON-OVERLAPPING.
 *
 * If the distance takes the traversal beyond the bounds, it returns the last valid point
 * within the bounds or the first valid point depending on direction.
 *
 * @template T
 * @param {Array<Crude<T>>} bounds - An array of crude bounds (array of length 2 or
 * objects with `lower` and `upper` properties).
 * @param {T} start - The starting point of the traversal.
 * @param {T} dist - The distance to traverse. Positive values move forwards, and
 * negative values move backwards.
 *
 * Edge Cases:
 *
 * 1. **Traversal beyond the last bound**: If the traversal moves beyond the last
 * bound (in either direction), the traversal ends at the last valid position within
 * the bounds.
 *    - Example: `traverse([[0, 10], [20, 30]], 25, 10); // => 30`
 *      (stops at the upper limit of the last bound)
 *
 * 2. **Traversal from a point outside the bounds**: If the starting point is outside
 *      the bounds and the traversal distance would move within bounds, it finds the
 *      closest bound and continues traversal from there.
 *    - Example: `traverse([[0, 10], [20, 30]], 15, 5); // => 25` (enters the second bound)
 *
 * 3. **Distance of 0**: If the distance is `0`, the traversal will return the starting
 *      point without moving.
 *    - Example: `traverse([[0, 10], [20, 30]], 5, 0); // => 5`
 *
 * @returns {T} The end point of the traversal within the bounds.
 *
 * @example
 * // Traversing 5 units forward from 5, ending exactly at the upper bound of the first
 * range.
 * traverse([[0, 10], [20, 30]], 5, 5);
 * // => 10
 *
 * @example
 * // Traversing 10 units forward from 5, crossing from the first range to the second.
 * traverse([[0, 10], [20, 30]], 5, 10);
 * // => 25
 *
 * @example
 * // Traversing 5 units forward starting outside the bounds, the traversal enters the
 * // second bound.
 * traverse([[0, 10], [20, 30]], 15, 5);
 * // => 25
 *
 * @example
 * // Traversing 30 units forward, stopping at the upper end of the second bound.
 * traverse([[0, 10], [20, 30]], 15, 30);
 * // => 30
 *
 * @example
 * // Traversing 7 units backward starting from 17, moving into the first bound.
 * traverse([[0, 5], [5, 10], [15, 20]], 17, -7);
 * // => 5
 *
 * @example
 * // Traversing beyond the last bound in a positive direction.
 * traverse([[0, 10], [20, 30]], 25, 10);
 * // => 30 (stops at the upper limit of the last bound)
 *
 * @example
 * // Traversing backward from a point not within any bound.
 * traverse([[0, 5], [10, 15]], 20, -10);
 * // => 15 (stops at the upper limit of the nearest previous bound)
 *
 * @example
 * // Traversing a distance of 0 from a point returns the starting point.
 * traverse([[0, 10], [20, 30]], 5, 0);
 * // => 5
 *
 * @throws {Error} If invalid bounds are provided, such as bounds arrays not being of
 * length 2.
 *
 * See {@link construct} for constructing valid bounds.
 *
 */
export const traverse = <T extends numeric.Value = number>(
  bounds: Array<Crude<T>>,
  start: T,
  dist: T,
): T => {
  const _bounds = bounds.map((b) => construct(b));

  const dir = dist > 0 ? 1 : dist < 0 ? -1 : 0;

  // If there's no distance to traverse, return the starting point
  if (dir === 0) return start;

  let remainingDist = dist;
  let currentPosition = start as number | bigint;

  while (mathEqual(remainingDist, 0) === false) {
    // Find the bound we're currently in or adjacent to
    const index = _bounds.findIndex((b) => {
      if (dir > 0) return currentPosition >= b.lower && currentPosition < b.upper;
      return currentPosition > b.lower && currentPosition <= b.upper;
    });

    if (index !== -1) {
      const b = _bounds[index];
      let distanceInBound: T;
      if (dir > 0) distanceInBound = sub(b.upper, currentPosition);
      else distanceInBound = sub(currentPosition, b.lower) as T;

      if (distanceInBound > (0 as T)) {
        const moveDist = mathMin(abs(remainingDist), distanceInBound);
        currentPosition = add(currentPosition, dir > 0 ? moveDist : -moveDist) as T;
        remainingDist = sub<T>(remainingDist, dir > 0 ? moveDist : -moveDist);

        // If we've exhausted the distance, return the current position
        if (mathEqual(remainingDist, 0)) return currentPosition as T;
        continue;
      }
    }

    // If we're not inside any bound, or we've reached the boundary
    if (dir > 0) {
      // Move to the next bound's lower value
      const nextBounds = _bounds.filter((b) => b.lower > currentPosition);
      if (nextBounds.length > 0) currentPosition = nextBounds[0].lower;
      // No more bounds in this direction
      else return currentPosition as T;
    } else {
      // Move to the previous bound's upper value
      const prevBounds = _bounds.filter((b) => b.upper < currentPosition);
      if (prevBounds.length > 0)
        currentPosition = prevBounds[prevBounds.length - 1].upper;
      // No more bounds in this direction
      else return currentPosition as T;
    }
  }
  return currentPosition as T;
};

/**
 * Returns the number of values within the given bounds, 'skip'ing over values that are
 * not within the bounds.
 *
 * Crucially, this function assumes that the bounds are ORDERED and NON-OVERLAPPING.
 *
 * @example
 * bounds.distance(
 *  [[0, 10], [20, 30]]
 *  5,
 *  5,
 * ) // => 0
 *
 * @example
 * bounds.distance(
 * [[0, 10], [20, 30]]
 * 5,
 * 25,
 * ) // => 10
 *
 * @example
 * bounds.distance(
 * [[0, 10], [20, 30]]
 * 15,
 * 25,
 * ) // => 5
 *
 * @example
 * bounds.distance(
 * [[0, 10], [20, 30]]
 * 15,
 * 5,
 * ) // => 5
 *
 * @param bounds
 * @param a - The start value.
 * @param b  - The end value.
 */
export const distance = <T extends numeric.Value = number>(
  bounds: Array<Crude<T>>,
  a: T,
  b: T,
): T => {
  const _bounds = bounds.map((b) => construct<T>(b));

  // If start and end are the same, the distance is zero
  if (a === b) return (typeof a === "bigint" ? 0n : 0) as T;

  // Determine the interval between a and b
  const interval = a < b ? construct([a, b]) : construct([b, a]);

  let totalDistance: T = (typeof a === "bigint" ? 0n : 0) as T;

  for (const bound of _bounds) {
    // Find the overlap between the interval and the current bound
    const overlapLower = bound.lower > interval.lower ? bound.lower : interval.lower;
    const overlapUpper = bound.upper < interval.upper ? bound.upper : interval.upper;

    // If there is an overlap, add its span to the total distance
    if (overlapLower < overlapUpper) {
      const overlapSpan = (overlapUpper - overlapLower) as T;
      // @ts-expect-error - typescript doesn't recognize that totalDistance is a number
      totalDistance = (totalDistance + overlapSpan) as T;
    }
  }

  return totalDistance;
};

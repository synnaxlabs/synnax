// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@/math";
import { type Bounds, bounds, type CrudeBounds } from "@/spatial/base";

export { type Bounds, bounds };

export type Crude<T extends number | bigint = number> = CrudeBounds<T>;

export const construct = <T extends bigint | number>(
  lower: T | Crude<T>,
  upper?: T,
): Bounds<T> => {
  const b: Bounds<T> = {} as const as Bounds<T>;
  if (typeof lower === "number" || typeof lower === "bigint") {
    if (upper != null) {
      b.lower = lower;
      b.upper = upper;
    } else {
      b.lower = (typeof lower === "bigint" ? 0n : 0) as T;
      b.upper = lower;
    }
  } else if (Array.isArray(lower)) [b.lower, b.upper] = lower;
  else {
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
};

export const makeValid = <T extends number | bigint>(a: Bounds<T>): Bounds<T> => {
  if (a.lower > a.upper) return { lower: a.upper, upper: a.lower };
  return a;
};

export const clamp = <T extends number | bigint>(bounds: Crude<T>, target: T): T => {
  const _bounds = construct<T>(bounds);
  if (target < _bounds.lower) return _bounds.lower;
  if (target >= _bounds.upper)
    return (_bounds.upper - ((typeof _bounds.upper === "number" ? 1 : 1n) as T)) as T;
  return target;
};

export const contains = <T extends bigint | number>(
  bounds: Crude<T>,
  target: T | CrudeBounds<T>,
): boolean => {
  const _bounds = construct(bounds);
  if (typeof target === "number" || typeof target === "bigint")
    return target >= _bounds.lower && target < _bounds.upper;
  const _target = construct(target);
  return _target.lower >= _bounds.lower && _target.upper <= _bounds.upper;
};

export const overlapsWith = <T extends bigint | number>(
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

export const span = <T extends number | bigint>(a: Crude<T>): T => {
  const _a = construct<T>(a);
  return (_a.upper - _a.lower) as T;
};

export const isZero = <T extends number | bigint>(a: Crude<T>): boolean => {
  const _a = construct(a);
  if (typeof _a.lower === "bigint") return _a.lower === 0n && _a.upper === 0n;
  return _a.lower === 0 && _a.upper === 0;
};

export const spanIsZero = (a: Crude): boolean => span(a) === 0;

export const isFinite = (a: Crude): boolean => {
  const _a = construct(a);
  return Number.isFinite(_a.lower) && Number.isFinite(_a.upper);
};

export const max = (bounds: Crude[]): Bounds => ({
  lower: Math.min(...bounds.map((b) => construct(b).lower)),
  upper: Math.max(...bounds.map((b) => construct(b).upper)),
});

export const min = (bounds: Crude[]): Bounds => ({
  lower: Math.max(...bounds.map((b) => construct(b).lower)),
  upper: Math.min(...bounds.map((b) => construct(b).upper)),
});

export const constructArray = <T extends bigint | number = number>(
  bounds: Crude<T>,
): T[] => {
  const _bounds = construct(bounds);
  const isBigInt = typeof _bounds.lower === "bigint";
  return Array.from({ length: Number(span(bounds)) }, (_, i) => {
    if (isBigInt) return ((_bounds.lower as bigint) + BigInt(i)) as T;
    return (_bounds.lower as number) + i;
  }) as T[];
};

export const findInsertPosition = <T extends bigint | number>(
  bounds: Array<Crude<T>>,
  target: T,
): { index: number; position: number } => {
  const _bounds = bounds.map((b) => construct<T>(b));
  const index = _bounds.findIndex(
    (b, i) => contains<T>(b, target) || target < _bounds[i].lower,
  );
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
export const buildInsertionPlan = <T extends number | bigint>(
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
  let removeBefore = math.sub(Number(span(_bounds[lower.index])), lower.position);
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

export const insert = <T extends number | bigint = number>(
  bounds: Array<Crude<T>>,
  value: Crude<T>,
): Array<Bounds<T>> => {
  const plan = buildInsertionPlan(bounds, value);
  const out = bounds.map((b) => construct(b));
  if (plan == null) return out;
  const _target = construct(value);
  _target.lower = math.add(_target.lower, plan.removeBefore);
  _target.upper = math.sub(_target.upper, plan.removeAfter);
  out.splice(plan.insertInto, plan.deleteInBetween, _target);
  return out;
};

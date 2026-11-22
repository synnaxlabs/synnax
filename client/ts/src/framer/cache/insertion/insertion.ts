// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type numeric } from "@synnaxlabs/x";

/**
 * Finds where target belongs in an ordered, non-overlapping array of bounds.
 *
 * @returns `index`, the bounds entry the target falls in, or where a new entry would
 * be inserted when it falls outside every one; and `position`, the offset of the
 * target inside that entry, or 0 when it falls outside.
 */
const findInsertPosition = <T extends numeric.Value>(
  ordered: Array<bounds.Crude<T>>,
  target: T,
): { index: number; position: number } => {
  const _ordered = ordered.map((b) => bounds.construct<T>(b));
  const index = _ordered.findIndex(
    (b) => bounds.contains<T>(b, target) || target < b.lower,
  );
  if (index === -1) return { index: ordered.length, position: 0 };
  const b = _ordered[index];
  if (bounds.contains(b, target)) return { index, position: Number(target - b.lower) };
  return { index, position: 0 };
};

/** A plan for splicing a new bound into an ordered array of bounds. */
interface InsertionPlan {
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
  ordered: Array<bounds.Crude<T>>,
  value: bounds.Crude<T>,
): InsertionPlan | null => {
  const _ordered = ordered.map((b) => bounds.construct<T>(b));
  const _target = bounds.construct(value);
  if (_ordered.length === 0) return ZERO_PLAN;
  if (_ordered.some((b) => bounds.contains(b, _target))) return null;
  const lower = findInsertPosition<T>(_ordered, _target.lower);
  const upper = findInsertPosition<T>(_ordered, _target.upper);
  // Target starts past every bound.
  if (lower.index === ordered.length)
    return { ...ZERO_PLAN, insertInto: ordered.length };
  if (lower.index === upper.index)
    return {
      removeAfter: upper.position,
      removeBefore: 0,
      insertInto: lower.index,
      deleteInBetween: 0,
    };
  let deleteInBetween = upper.index - lower.index;
  let insertInto = lower.index;
  let removeBefore = Number(bounds.span(_ordered[lower.index])) - lower.position;
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

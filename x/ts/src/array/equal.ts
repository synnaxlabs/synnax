// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** @returns true if a and b have the same length and each element is ===. */
export const equal = <T>(a: readonly T[], b: readonly T[]): boolean => {
  if (a === b) return true;
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
};

/**
 * @returns true if a and b hold the same elements with the same counts, in any order.
 * Elements match as Map keys do, so NaN matches NaN.
 */
export const equalUnordered = <T>(a: readonly T[], b: readonly T[]): boolean => {
  if (a.length !== b.length) return false;
  const counts = new Map<T, number>();
  for (const v of a) counts.set(v, (counts.get(v) ?? 0) + 1);
  for (const v of b) {
    const count = counts.get(v);
    if (count == null || count === 0) return false;
    counts.set(v, count - 1);
  }
  return true;
};

/**
 * @returns true if a and b hold the same distinct elements, ignoring order and
 * duplicates. Elements match as Set members do, so NaN matches NaN.
 */
export const equalAsSets = <T>(a: readonly T[], b: readonly T[]): boolean => {
  const setA = new Set(a);
  const setB = new Set(b);
  if (setA.size !== setB.size) return false;
  for (const v of setA) if (!setB.has(v)) return false;
  return true;
};

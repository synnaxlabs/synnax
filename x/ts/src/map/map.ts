// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * @returns the value at key. When the map holds no value at key, sets key to the result
 * of create and returns that result. A stored undefined counts as no value.
 */
export const getOrInsert = <K, V>(
  map: Map<K, V>,
  key: K,
  create: () => NoInfer<V>,
): V => {
  const value = map.get(key);
  if (value !== undefined) return value;
  const created = create();
  map.set(key, created);
  return created;
};

/**
 * Adds by to the count at key, treating a missing key as zero.
 * @returns the new count.
 */
export const increment = <K>(map: Map<K, number>, key: K, by: number = 1): number => {
  const count = (map.get(key) ?? 0) + by;
  map.set(key, count);
  return count;
};

/** @returns true if a and b have the same keys and the value of each key is ===. */
export const equal = <K, V>(a: Map<K, V>, b: Map<K, V>): boolean => {
  if (a === b) return true;
  if (a.size !== b.size) return false;
  for (const [k, v] of a) if (!b.has(k) || b.get(k) !== v) return false;
  return true;
};

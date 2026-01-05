// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Removes duplicate values from an array, preserving the order of the first occurrence
 * of each unique value.
 *
 * @param values - An array or readonly array of values to deduplicate.
 * @returns A new array containing only unique values.
 *
 * @example
 * ```typescript
 * unique([1, 2, 2, 3, 4, 4, 5]); // [1, 2, 3, 4, 5]
 * ```
 */
export const unique = <V>(values: V[] | readonly V[]): V[] => [...new Set(values)];

/**
 * Removes duplicate values from an array based on a key function, preserving either
 * the first or last occurrence of each unique key. If
 *
 * @param values - An array or readonly array of values to deduplicate.
 * @param key - A function that creates a unique key for each value.
 * @param keepFirst - An optional boolean indicating whether to keep the first instance
 *                    (`true`, default) or the last instance (`false`) of each unique key.
 * @returns A new array containing only unique values based on the created keys.
 *
 * @example
 * // Default behavior (keep first instance):
 * by(
 *   [{ id: 1, name: "A" }, { id: 2, name: "B" }, { id: 1, name: "C" }],
 *   (value) => value.id
 * );
 * // Result: [{ id: 1, name: "A" }, { id: 2, name: "B" }]
 *
 * @example
 * // Keep last instance:
 * by(
 *   [{ id: 1, name: "A" }, { id: 2, name: "B" }, { id: 1, name: "C" }],
 *   (value) => value.id,
 *   false
 * );
 * // Result: [{ id: 2, name: "B" }, { id: 1, name: "C" }]
 */
export const by = <V>(
  values: V[] | readonly V[],
  key: (value: V) => unknown,
  keepFirst: boolean = true,
): V[] => {
  const map = new Map<unknown, V>();
  values.forEach((v) => {
    const k = key(v);
    if (map.has(k)) {
      if (keepFirst) return;
      map.delete(k);
    }
    // different delete and set operations for keepLast so order is preserved
    map.set(k, v);
  });
  return Array.from(map.values());
};

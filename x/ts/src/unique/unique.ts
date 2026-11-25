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
 * @param values - An array or readonly array of values to deduplicate.
 * @returns A new array containing only unique values.
 *
 * @example
 * ```typescript
 * unique([1, 2, 2, 3, 4, 4, 5]); // [1, 2, 3, 4, 5]
 * ```
 */
export const unique = <V>(values: V[] | readonly V[]): V[] => [...new Set(values)];

/** Options for {@link by}. */
export interface ByOptions {
  /** Which instance of each key to keep. Defaults to "first". */
  keep?: "first" | "last";
}

interface By {
  /**
   * Removes duplicate values from an array based on a key function, preserving the
   * order of the instances it keeps.
   * @param values - An array or readonly array of values to deduplicate.
   * @param key - Returns the key that two values must share to be duplicates.
   * @returns A new array containing only unique values based on the created keys.
   *
   * @example
   * // Keep the last instance:
   * by(
   *   [{ id: 1, name: "A" }, { id: 2, name: "B" }, { id: 1, name: "C" }],
   *   (value) => value.id,
   *   { keep: "last" },
   * );
   * // Result: [{ id: 2, name: "B" }, { id: 1, name: "C" }]
   */
  <V>(values: V[] | readonly V[], key: (value: V) => unknown, options?: ByOptions): V[];
  /** @deprecated Pass `{ keep: "last" }` in place of false. Removed in 0.64. */
  <V>(
    values: V[] | readonly V[],
    key: (value: V) => unknown,
    keepFirst: boolean | undefined,
  ): V[];
}

export const by: By = <V>(
  values: V[] | readonly V[],
  key: (value: V) => unknown,
  options: ByOptions | boolean = {},
): V[] => {
  const keepFirst =
    typeof options === "boolean" ? options : (options.keep ?? "first") === "first";
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

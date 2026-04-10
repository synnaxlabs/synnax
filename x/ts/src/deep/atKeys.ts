// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface AtKeysResult<V = unknown> {
  /** True if every key in the path resolved to a property that is present on its
   * container (via `in`), false if any segment was missing. A key with an explicit
   * `undefined` or `null` value is still considered present. */
  present: boolean;
  /** The value at the path. Undefined when `present` is false. */
  value: V | undefined;
}

/**
 * Walks a value by a strict `PropertyKey[]` path and returns whether the path is
 * present along with the value at that location.
 *
 * Differs from {@link get}/{@link has} in three important ways:
 *
 * 1. Takes a `PropertyKey[]` instead of a dotted string, so keys containing literal
 *    `.` are handled unambiguously.
 * 2. Uses plain positional key lookup (`container[key]`) without the
 *    `defaultGetter` fallback that searches arrays for items with matching `.key`.
 * 3. Distinguishes "key missing from container" from "key present with
 *    `null`/`undefined` value" by using the `in` operator, which `has` does not.
 *
 * Intended for consumers that receive paths as `PropertyKey[]` (e.g. zod issues,
 * JSON-pointer-like walkers) and need to render the exact state of the input,
 * including present-but-null fields, without the ergonomic heuristics of `get`.
 */
export const atKeys = <V = unknown>(
  root: unknown,
  path: ReadonlyArray<PropertyKey>,
): AtKeysResult<V> => {
  let cur: unknown = root;
  for (const key of path) {
    if (cur == null || typeof cur !== "object")
      return { present: false, value: undefined };
    const container = cur as Record<PropertyKey, unknown>;
    if (!(key in container)) return { present: false, value: undefined };
    cur = container[key];
  }
  return { present: true, value: cur as V };
};

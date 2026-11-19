// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { text } from "@/compare/text";

/**
 * Orders two values: a negative number if a orders before b, a positive number if
 * after, and 0 if they order the same.
 */
export type Comparator<T> = (a: T, b: T) => number;

/**
 * Orders numbers in ascending order. The result is NaN when a or b is NaN, so a sort
 * of an array that holds NaN has no defined order.
 */
export const numbers: Comparator<number> = (a, b) => a - b;

/** Orders bigints in ascending order. */
export const bigints: Comparator<bigint> = (a, b) => (a < b ? -1 : a > b ? 1 : 0);

export interface By {
  <T>(key: (v: T) => string): Comparator<T>;
  <T, K>(key: (v: T) => K, cmp: Comparator<K>): Comparator<T>;
}

/**
 * Orders values by a key taken from each one.
 * @param key - Takes the key from a value.
 * @param cmp - Orders the keys. Defaults to {@link text}.
 */
export const by: By =
  <T, K>(key: (v: T) => K, cmp: Comparator<K> = text as Comparator<K>): Comparator<T> =>
  (a, b) =>
    cmp(key(a), key(b));

/** @returns a comparator that orders values in the opposite order of cmp. */
export const reverse =
  <T>(cmp: Comparator<T>): Comparator<T> =>
  (a, b) =>
    cmp(b, a);

/**
 * @returns a comparator that returns the first non-zero result of cmps, in order, or 0
 * when all of them return 0.
 */
export const chain =
  <T>(...cmps: Array<Comparator<T>>): Comparator<T> =>
  (a, b) => {
    for (const cmp of cmps) {
      const c = cmp(a, b);
      if (c !== 0) return c;
    }
    return 0;
  };

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Comparator, newF } from "@/compare/compare";

/**
 * Performs a binary search on the given array. If the array is not sorted, or the
 * comparator does not correctly order the values, the behavior of this function is
 * undefined. If the value is not found, the index of the first element greater than the
 * value is returned. If the value is greater than all elements in the array, the length
 * of the array is returned.
 *
 * @param array - The array to search.
 * @param value - The value to search for.
 * @param comparator - The comparator to use.
 * @returns The index of the value in the array.
 */
export const search = <T>(array: T[], value: T, comparator?: Comparator<T>): number => {
  let left = 0;
  let right = array.length;
  const cf = comparator ?? newF(value);
  while (left < right) {
    const mid = Math.floor((left + right) / 2);
    const cmp = cf(array[mid], value);
    if (cmp === 0) return mid;
    if (cmp < 0) left = mid + 1;
    else right = mid;
  }
  return left;
};

/**
 * Inserts a value into the given sorted array and maintains the sort order of the
 * array. If the array is not sorted, or the comparator does not correctly order the
 * values, the behavior of this function is undefined.
 *
 * @param array - The array to insert the value into.
 * @param value - The value to insert.
 * @param comparator - The comparator to use.
 */
export const insert = <T>(array: T[], value: T, comparator?: Comparator<T>): void => {
  const idx = search(array, value, comparator);
  array.splice(idx, 0, value);
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Converts a value to an array, handling null and undefined values. If the input is already an array,
 * returns the same array reference. If the input is null or undefined, returns an empty array.
 * If the input is a single value, wraps it in an array.
 *
 * @template T - The type of the input value
 * @param value - The value to convert to an array
 * @returns An array containing the input value, an empty array for null/undefined, or the input array if it was already an array
 *
 * @example
 * ```ts
 * nullToArray(1) // returns [1]
 * nullToArray(null) // returns []
 * nullToArray(undefined) // returns []
 * nullToArray([1, 2, 3]) // returns [1, 2, 3]
 * ```
 */
export const toArray = <T>(value: T | T[] | null | undefined): T[] =>
  Array.isArray(value) ? value : value == null ? [] : [value];

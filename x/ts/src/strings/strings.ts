// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Joins an array of strings into a natural language list.
 *
 * @param strings - The array of strings to join.
 * @param zeroLength - The string to return if the array is empty. Defaults to an empty string.
 * @returns A string that represents the natural language list.
 *
 * @example
 * ```typescript
 * naturalLanguageJoin([]); // ""
 * naturalLanguageJoin([], "No items"); // "No items"
 * naturalLanguageJoin(["apple"]); // "apple"
 * naturalLanguageJoin(["apple", "banana"]); // "apple and banana"
 * naturalLanguageJoin(["apple", "banana", "cherry"]); // "apple, banana, and cherry"
 * ```
 */
export const naturalLanguageJoin = (
  strings: string[],
  zeroLength: string = "",
): string => {
  const length = strings.length;
  if (length === 0) return zeroLength;
  if (length === 1) return strings[0];
  if (length === 2) return `${strings[0]} and ${strings[1]}`;
  return `${strings.slice(0, -1).join(", ")}, and ${strings[length - 1]}`;
};

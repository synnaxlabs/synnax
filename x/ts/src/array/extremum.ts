// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const extremumBy = <T>(
  values: readonly T[],
  key: (value: T) => number,
  beats: (candidate: number, best: number) => boolean,
): T | undefined => {
  let best: { value: T; key: number } | undefined;
  for (const value of values) {
    const k = key(value);
    if (best == null || beats(k, best.key)) best = { value, key: k };
  }
  return best?.value;
};

/**
 * @returns the first value with the largest key, or undefined when values is empty.
 * Unlike Math.max(...values.map(key)), it does not spread, so it handles any length.
 */
export const maxBy = <T>(
  values: readonly T[],
  key: (value: T) => number,
): T | undefined => extremumBy(values, key, (candidate, best) => candidate > best);

/**
 * @returns the first value with the smallest key, or undefined when values is empty.
 * Unlike Math.min(...values.map(key)), it does not spread, so it handles any length.
 */
export const minBy = <T>(
  values: readonly T[],
  key: (value: T) => number,
): T | undefined => extremumBy(values, key, (candidate, best) => candidate < best);

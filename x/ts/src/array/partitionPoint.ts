// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Searches [0, n) for the first index where pred is true, in O(log n) calls to pred.
 * pred must be false for every index before that one and true for every index after
 * it.
 * @returns the first index where pred is true, or n when pred is true for no index.
 */
export const partitionPoint = (n: number, pred: (i: number) => boolean): number => {
  let lo = 0;
  let hi = n;
  while (lo < hi) {
    const mid = lo + ((hi - lo) >>> 1);
    if (pred(mid)) hi = mid;
    else lo = mid + 1;
  }
  return lo;
};

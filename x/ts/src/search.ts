// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CompareF } from "@/compare";
import { type Key, type KeyedRecord } from "@/record";

const binary = <T>(arr: T[], target: T, compare: CompareF<T>): number => {
  let left = 0;
  let right = arr.length - 1;
  while (left <= right) {
    const mid = Math.floor((left + right) / 2);
    const cmp = compare(arr[mid], target);
    if (cmp === 0) return mid;
    if (cmp < 0) left = mid + 1;
    else right = mid - 1;
  }
  return -1;
};

export const Search = {
  binary,
};

export interface TermSearcher<T, K extends Key, E extends KeyedRecord<K, E>> {
  search: (term: T) => E[];
  retrieve: (keys: K[]) => E[];
}

export interface AsyncTermSearcher<T, K extends Key, E extends KeyedRecord<K, E>> {
  search: (term: T) => Promise<E[]>;
  retrieve: (keys: K[]) => Promise<E[]>;
}

// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * @packageDocumentation
 * The search package provides utilities for searching and retrieving data, including
 * binary search implementation and interfaces for term-based searching.
 */

import { type compare } from "@/compare";
import { type observe } from "@/observe";
import { type Key, type Keyed } from "@/record";

/**
 * Performs a binary search on a sorted array.
 *
 * @param arr - The sorted array to search in
 * @param target - The value to search for
 * @param compare - A comparison function that returns:
 *                 - negative if a < b
 *                 - zero if a === b
 *                 - positive if a > b
 * @returns The index of the target element if found, -1 otherwise
 */
export const binary = <T>(
  arr: T[],
  target: T,
  compare: compare.Comparer<T>,
): number => {
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

/**
 * Interface for synchronous term-based searching operations.
 *
 * @typeParam T - The type of the search term
 * @typeParam K - The type of the key used to identify entities
 * @typeParam E - The type of the entity being searched, must implement Keyed<K>
 */
export interface TermSearcher<T, K extends Key, E extends Keyed<K>>
  extends Partial<observe.Observable<E[]>> {
  /** Identifier for the type of searcher */
  readonly type: string;
  /** Searches for entities matching the given term */
  search: (term: T) => E[];
  /** Retrieves entities by their keys */
  retrieve: (keys: K[]) => E[];
  /** Retrieves a paginated subset of entities */
  page: (offset: number, limit: number) => E[];
}

/**
 * Interface for asynchronous term-based searching operations.
 *
 * @typeParam T - The type of the search term
 * @typeParam K - The type of the key used to identify entities
 * @typeParam E - The type of the entity being searched, must implement Keyed<K>
 */
export interface AsyncTermSearcher<T, K extends Key, E extends Keyed<K>>
  extends Partial<observe.Observable<E[]>> {
  /** Identifier for the type of searcher */
  readonly type: string;
  /** Asynchronously searches for entities matching the given term */
  search: (term: T) => Promise<E[]>;
  /** Asynchronously retrieves entities by their keys */
  retrieve: (keys: K[]) => Promise<E[]>;
  /** Asynchronously retrieves a paginated subset of entities */
  page: (offset: number, limit: number) => Promise<E[]>;
}

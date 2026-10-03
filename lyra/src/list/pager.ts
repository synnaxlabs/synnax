// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type state } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

/**
 * Parameters for pagination functionality.
 * These parameters are automatically managed by the pager utilities.
 */
export type PagerParams = {
  /** Search term for filtering results */
  searchTerm?: string;
  /** Number of items to skip (for pagination) */
  offset?: number;
  /** Maximum number of items to return per page */
  limit?: number;
};

/** Return type for the usePager hook, providing pagination utilities. */
export interface UsePagerReturn {
  /** Function to fetch the next page of results */
  fetchMore: () => void;
  /** Function to perform a search with the given term */
  search: (term: string) => void;
}

interface RetrieveOptions {
  mode?: "append" | "replace";
}

/** Arguments for the usePager hook. */
export interface UsePagerParams {
  /** Function to retrieve data */
  retrieve: (
    setter: state.SetArg<PagerParams, Partial<PagerParams>>,
    options?: RetrieveOptions,
  ) => void;
  /** Number of items per page (default: 10) */
  pageSize?: number;
}

const DEFAULT_PAGE_SIZE = 10;

/** @returns the params for the page after the given one. */
export const page = (
  { offset, searchTerm = "", ...prev }: PagerParams,
  pageSize: number = DEFAULT_PAGE_SIZE,
): PagerParams => ({
  ...prev,
  offset: (offset ?? -pageSize) + pageSize,
  limit: pageSize,
  searchTerm,
});

/** @returns the given params rewound to the first page of a new search term. */
export const search = (
  prev: PagerParams,
  searchTerm: string,
  pageSize: number = DEFAULT_PAGE_SIZE,
) => ({
  ...prev,
  searchTerm,
  offset: 0,
  limit: pageSize,
});

/**
 * Turns a flux list query's `retrieve` into paging and search callbacks, tracking the
 * offset itself. Wire `fetchMore` to the frame's `onFetchMore`.
 *
 * @example const { fetchMore, search } = List.usePager({ retrieve, pageSize: 20 });
 */
export const usePager = ({
  retrieve,
  pageSize = DEFAULT_PAGE_SIZE,
}: UsePagerParams): UsePagerReturn => {
  /** Fetches the next page of results by incrementing the offset. */
  const fetchMore = useCallback(() => {
    retrieve((prev) => page(prev, pageSize), { mode: "append" });
  }, [retrieve, pageSize]);

  /** Performs a search with the given term, resetting to the first page. */
  const handleSearch = useCallback(
    (searchTerm: string) => retrieve((prev) => search(prev, searchTerm, pageSize)),
    [retrieve, pageSize],
  );

  return useMemo(
    () => ({ fetchMore, search: handleSearch }),
    [fetchMore, handleSearch],
  );
};

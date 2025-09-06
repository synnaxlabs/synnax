// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { type state } from "@/state";

/**
 * Parameters for pagination functionality.
 * These parameters are automatically managed by the pager utilities.
 */
export interface PagerParams {
  /** Search term for filtering results */
  searchTerm?: string;
  /** Number of items to skip (for pagination) */
  offset?: number;
  /** Maximum number of items to return per page */
  limit?: number;
}

/**
 * Return type for the usePager hook, providing pagination utilities.
 */
export interface UsePagerReturn {
  /** Function to fetch the next page of results */
  fetchMore: () => void;
  /** Function to perform a search with the given term */
  search: (term: string) => void;
}

interface RetrieveOptions {
  mode?: "append" | "replace";
}

/**
 * Arguments for the usePager hook.
 */
export interface UsePagerArgs {
  /** Function to retrieve data */
  retrieve: (
    setter: state.SetArg<PagerParams, Partial<PagerParams>>,
    options?: RetrieveOptions,
  ) => void;
  /** Number of items per page (default: 10) */
  pageSize?: number;
}

/**
 * Hook that provides pagination utilities for list queries.
 *
 * This hook works with flux list queries to provide easy pagination and search
 * functionality. It automatically manages offset calculations and search term
 * handling.
 *
 * @param config Configuration object containing retrieve function and optional page size
 * @returns Object with pagination and search utilities
 *
 * @example
 * ```typescript
 * const listQuery = useList({ name: "users", retrieve: fetchUsers });
 * const { onFetchMore, onSearch } = usePager({
 *   retrieve: listQuery.retrieve,
 *   pageSize: 20
 * });
 *
 * // Fetch next page
 * onFetchMore();
 *
 * // Search for users
 * onSearch("john");
 * ```
 */
export const usePager = ({ retrieve, pageSize = 10 }: UsePagerArgs): UsePagerReturn => {
  /**
   * Fetches the next page of results by incrementing the offset.
   */
  const fetchMore = useCallback(() => {
    retrieve(
      ({ offset = -pageSize, searchTerm = "", ...prev }) => ({
        ...prev,
        offset: offset + pageSize,
        limit: pageSize,
        searchTerm,
      }),
      { mode: "append" },
    );
  }, [retrieve, pageSize]);

  /**
   * Performs a search with the given term, resetting to the first page.
   */
  const search = useCallback(
    (searchTerm: string) =>
      retrieve((prev) => ({ ...prev, searchTerm, offset: 0, limit: pageSize })),
    [retrieve, pageSize],
  );

  return { fetchMore, search };
};

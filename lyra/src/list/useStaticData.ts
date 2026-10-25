// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type compare, type record, type state } from "@synnaxlabs/x";
import Fuse from "fuse.js";
import { useMemo, useState } from "react";

import { type FrameProps, type GetItem } from "@/list/Frame";

/** Return value for {@link useStaticData}. Spread it into a {@link Frame}. */
export interface UseStaticDataReturn<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends Required<Pick<FrameProps<K, E>, "getItem">> {
  data: K[];
  retrieve: state.Setter<RetrieveParams, Partial<RetrieveParams>>;
}

/** Query the caller passes to `retrieve` to narrow the data. */
export interface RetrieveParams {
  searchTerm?: string;
  offset?: number;
  limit?: number;
}

export interface UseStaticDataParams<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> {
  data: readonly E[];
  /** Drops any item this rejects, before search and sort. */
  filter?: (item: E, params: RetrieveParams) => boolean;
  sort?: compare.Comparator<E>;
}

/**
 * Backs a {@link Frame} with an in-memory array, adding fuzzy search over every field
 * of the entry.
 *
 * @example <List.Frame {...List.useStaticData({ data: MODES })}>
 */
export const useStaticData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({
  data,
  filter,
  sort,
}: UseStaticDataParams<K, E>): UseStaticDataReturn<K, E> => {
  const [params, setParams] = useState<RetrieveParams>({});
  let filteredData = data;
  if (filter != null) filteredData = filteredData.filter((d) => filter(d, {}));
  if (sort != null) filteredData = [...filteredData].sort(sort);
  // The compiler reads fuse.search as a write to filteredData, so without this memo it
  // rebuilds the index and getItem on every search.
  // eslint-disable-next-line no-restricted-syntax
  const fuse = useMemo(() => {
    if (filteredData.length === 0) return null;
    return new Fuse(filteredData, {
      keys: Object.keys(filteredData[0]),
      threshold: 0.3,
    });
  }, [filteredData]);
  const getItem = ((key: K | K[]) => {
    if (Array.isArray(key)) {
      const keySet = new Set(key);
      return filteredData.filter((d) => keySet.has(d.key));
    }
    return filteredData.find((d) => d.key === key);
  }) as GetItem<K, E>;
  let processedData = filteredData;
  if (params.searchTerm != null && params.searchTerm.length > 0 && fuse != null) {
    const searchResults = fuse.search(params.searchTerm);
    processedData = searchResults.map((result) => result.item);
    if (sort != null) processedData = [...processedData].sort(sort);
  }
  const keys = processedData.map((d) => d.key);
  const res = { getItem, data: keys };
  return { ...res, retrieve: setParams };
};

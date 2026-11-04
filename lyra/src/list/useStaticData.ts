// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type compare, type record, search, type state } from "@synnaxlabs/x";
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

/** The text of each string or number field of entry, for the search index. */
const texts = (entry: object): string[] =>
  Object.values(entry)
    .filter((v) => typeof v === "string" || typeof v === "number")
    .map(String);

/**
 * Backs a {@link Frame} with an in-memory array, adding a search over the text of every
 * field of the entry.
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
  // The compiler reads index.search as a write to filteredData, so without this memo it
  // rebuilds the index and getItem on every search.
  // eslint-disable-next-line no-restricted-syntax
  const index = useMemo(() => {
    const index = new search.Index<K>();
    for (const entry of filteredData) index.set(entry.key, ...texts(entry));
    return index;
  }, [filteredData]);
  const getItem = ((key: K | K[]) => {
    if (Array.isArray(key)) {
      const keySet = new Set(key);
      return filteredData.filter((d) => keySet.has(d.key));
    }
    return filteredData.find((d) => d.key === key);
  }) as GetItem<K, E>;
  let keys = filteredData.map((d) => d.key);
  if (params.searchTerm != null && params.searchTerm.length > 0) {
    const found = index.search(params.searchTerm);
    // keys are in sort order already, so a sort wins over the match rank.
    if (sort == null) keys = found;
    else {
      const foundSet = new Set(found);
      keys = keys.filter((k) => foundSet.has(k));
    }
  }
  const res = { getItem, data: keys };
  return { ...res, retrieve: setParams };
};

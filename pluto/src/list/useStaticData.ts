// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type compare, type record } from "@synnaxlabs/x";
import Fuse from "fuse.js";
import { useCallback, useMemo, useState } from "react";

import { type FrameProps, type GetItem } from "@/list/Frame";
import { type state } from "@/state";

export interface UseStaticDataReturn<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends Required<Pick<FrameProps<K, E>, "getItem">> {
  data: K[];
  retrieve: state.Setter<RetrieveParams, Partial<RetrieveParams>>;
}

export interface RetrieveParams {
  searchTerm?: string;
  offset?: number;
  limit?: number;
}

export interface UseStaticDataArgs<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> {
  data: E[];
  filter?: (item: E, params: RetrieveParams) => boolean;
  sort?: compare.Comparator<E>;
}

export const useStaticData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({
  data,
  filter,
  sort,
}: UseStaticDataArgs<K, E>): UseStaticDataReturn<K, E> => {
  const filteredData = useMemo(() => {
    let result = data;
    if (filter != null) result = result.filter((d) => filter(d, {}));
    if (sort != null) result = [...result].sort(sort);
    return result;
  }, [data, filter, sort]);
  const fuse = useMemo(() => {
    if (filteredData.length === 0) return null;
    return new Fuse(filteredData, {
      keys: Object.keys(filteredData[0]),
      threshold: 0.3,
    });
  }, [filteredData]);
  const [params, setParams] = useState<RetrieveParams>({});
  const getItem = useCallback(
    ((key: K | K[]) => {
      if (Array.isArray(key)) {
        const keySet = new Set(key);
        return filteredData.filter((d) => keySet.has(d.key));
      }
      return filteredData.find((d) => d.key === key);
    }) as GetItem<K, E>,
    [filteredData],
  );
  const res = useMemo(() => {
    let processedData = filteredData;
    if (params.searchTerm != null && params.searchTerm.length > 0 && fuse != null) {
      const searchResults = fuse.search(params.searchTerm);
      processedData = searchResults.map((result) => result.item);
      if (sort != null) processedData = [...processedData].sort(sort);
    }
    const keys = processedData.map((d) => d.key);
    return { getItem, data: keys };
  }, [filteredData, params, getItem, fuse, sort]);
  return useMemo(() => ({ ...res, retrieve: setParams }), [res, setParams]);
};

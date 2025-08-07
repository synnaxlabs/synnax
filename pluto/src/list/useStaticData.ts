// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import Fuse from "fuse.js";
import { useCallback, useMemo, useState } from "react";

import { type FrameProps } from "@/list/Frame";
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
}

export const useStaticData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({
  data,
  filter,
}: UseStaticDataArgs<K, E>): UseStaticDataReturn<K, E> => {
  const filteredData = useMemo(() => {
    if (filter == null) return data;
    return data.filter((d) => filter(d, {}));
  }, [data, filter]);
  const fuse = useMemo(() => {
    if (filteredData.length === 0) return null;
    return new Fuse(filteredData, {
      keys: Object.keys(filteredData[0]),
      threshold: 0.3,
    });
  }, [filteredData]);
  const [params, setParams] = useState<RetrieveParams>({});
  const getItem = useCallback(
    (key?: K) => filteredData.find((d) => d.key === key),
    [filteredData],
  );
  const res = useMemo(() => {
    let keys = filteredData.map((d) => d.key);
    if (params.searchTerm != null && params.searchTerm.length > 0 && fuse != null)
      keys = fuse.search(params.searchTerm).map((d) => d.item.key);
    return { getItem, data: keys };
  }, [filteredData, params, getItem, fuse]);
  return { ...res, retrieve: setParams };
};

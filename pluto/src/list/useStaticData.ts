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

import { type state } from "@/state";

export interface UseStaticDataReturn<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  useItem: (key?: K) => E | undefined;
  data: K[];
  retrieve: state.Setter<RetrieveParams, RetrieveParams | {}>;
}

export interface RetrieveParams {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useStaticData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>(
  data: E[],
): UseStaticDataReturn<K, E> => {
  const fuse = useMemo(
    () =>
      new Fuse(data, {
        keys: Object.keys(data[0]),
        threshold: 0.3,
      }),
    [data],
  );
  const [params, setParams] = useState<RetrieveParams>({});

  const res = useMemo(() => {
    const keys = fuse.search(params.term ?? "").map((d) => d.item.key);
    const useItem = useCallback((key?: K) => data.find((d) => d.key === key), [data]);
    return { useItem, data: keys };
  }, [data, params]);
  return { ...res, retrieve: setParams };
};

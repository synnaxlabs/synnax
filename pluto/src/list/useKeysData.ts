// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { type List } from "@/list";
import { type GetItem } from "@/list/Frame";

export interface UseKeysDataReturn<K extends record.Key = record.Key>
  extends Required<Pick<List.FrameProps<K, record.Keyed<K>>, "getItem">> {
  data: K[];
}

export const useKeysData = <K extends record.Key = record.Key>(
  data: K[] | readonly K[],
): UseKeysDataReturn<K> => {
  const getItem = useCallback(
    ((key: K | K[]) => {
      if (Array.isArray(key)) {
        const keys = new Set(key);
        return data.filter((d) => keys.has(d)).map((d) => ({ key: d }));
      }
      const option = data.find((option) => option === key);
      return option ? { key: option } : undefined;
    }) as GetItem<K, record.Keyed<K>>,
    [data],
  );
  return useMemo(() => ({ data: data as K[], getItem }), [data, getItem]);
};

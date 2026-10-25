// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { destructor, type record } from "@synnaxlabs/x";

import { type FrameProps, type GetItem } from "@/list/Frame";

export interface UseCombinedDataParams<
  K extends record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  first: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
  second: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
}

/**
 * Concatenates two list data sources into one, `first` before `second`. Reads and
 * subscriptions hit both.
 */
export const useCombinedData = <
  K extends record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  first,
  second,
}: UseCombinedDataParams<K, E>): FrameProps<K, E> => {
  const { data: firstData, getItem: firstGetItem, subscribe: firstSubscribe } = first;
  const {
    data: secondData,
    getItem: secondGetItem,
    subscribe: secondSubscribe,
  } = second;
  const data = [...firstData, ...secondData];
  const getItem = ((key: K | K[]) => {
    if (Array.isArray(key)) {
      const firstGotten = firstGetItem?.(key) ?? [];
      const secondGotten = secondGetItem?.(key) ?? [];
      return [...firstGotten, ...secondGotten];
    }
    return firstGetItem?.(key) ?? secondGetItem?.(key);
  }) as GetItem<K, E>;
  const subscribe = (callback: () => void, key: K) => {
    const firstUnsub = firstSubscribe?.(callback, key);
    const secondUnsub = secondSubscribe?.(callback, key);
    return () => destructor.unwind(firstUnsub, secondUnsub);
  };
  return { data, getItem, subscribe };
};

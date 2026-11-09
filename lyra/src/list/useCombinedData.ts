// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record } from "@synnaxlabs/x";

import { type FrameProps, type GetItem } from "@/list/Frame";

export interface UseCombinedDataParams<
  K extends record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  first: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
  second: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
}

const noop = () => {};

/**
 * Concatenates two list data sources into one, `first` before `second`. Reads try both.
 * A subscription goes to `first` for a key it lists, and to `second` for any other key,
 * since a remote source fetches a key it does not hold.
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
  const firstKeys = new Set(firstData);
  const subscribe = (callback: () => void, key: K): destructor.Destructor =>
    (firstKeys.has(key) ? firstSubscribe : secondSubscribe)?.(callback, key) ?? noop;
  return { data, getItem, subscribe };
};

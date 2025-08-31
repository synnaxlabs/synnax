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

import { type FrameProps } from "@/list/Frame";

export interface UseCombinedDataArgs<
  K extends record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  first: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
  second: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
}

export const useCombinedData = <
  K extends record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  first,
  second,
}: UseCombinedDataArgs<K, E>): FrameProps<K, E> => {
  const data = useMemo(
    () => [...first.data, ...second.data],
    [first.data, second.data],
  );
  const getItem = useCallback(
    (key: K) => first.getItem?.(key) ?? second.getItem?.(key),
    [first.getItem, second.getItem],
  );
  const subscribe = useCallback(
    (callback: () => void, key: K) => {
      const firstUnsub = first.subscribe?.(callback, key);
      const secondUnsub = second.subscribe?.(callback, key);
      return () => {
        firstUnsub?.();
        secondUnsub?.();
      };
    },
    [first.subscribe, second.subscribe],
  );
  return { data, getItem, subscribe };
};

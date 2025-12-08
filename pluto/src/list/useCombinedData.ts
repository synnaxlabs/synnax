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

import { type FrameProps, type GetItem } from "@/list/Frame";

export interface UseCombinedDataArg<
  K extends record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe"> {}

export const useCombinedData = <
  K extends record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(
  ...args: UseCombinedDataArg<K, E>[]
): FrameProps<K, E> => {
  const argsToData = useMemo(() => args.map(({ data }) => data), [args]);
  const data = useMemo(() => argsToData.flat(), [...argsToData]);
  const argsGetItem = useMemo(() => args.map(({ getItem }) => getItem), [args]);
  const getItem = useCallback(
    (key: K | K[]) => {
      if (Array.isArray(key))
        return argsGetItem.map((getItem) => getItem?.(key) ?? []).flat();
      return argsGetItem.find((getItem) => getItem?.(key) != null)?.(key);
    },
    [...argsGetItem],
  ) as GetItem<K, E>;
  const argsSubscribe = useMemo(() => args.map(({ subscribe }) => subscribe), [args]);
  const subscribe = useCallback(
    (callback: () => void, key: K) => {
      const unsubscribers = argsSubscribe.map((subscribe) =>
        subscribe?.(callback, key),
      );
      return () => unsubscribers.forEach((unsubscribe) => unsubscribe?.());
    },
    [...argsSubscribe],
  );
  return useMemo(() => ({ data, getItem, subscribe }), [data, getItem, subscribe]);
};

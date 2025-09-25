// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type record } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { useInitializerRef } from "@/hooks";
import { type FrameProps } from "@/list/Frame";

interface GetItem<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> {
  (key?: K): E | undefined;
  (key: K[]): E[];
}

export interface UseMapDataReturn<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> extends Required<Pick<FrameProps<K, E>, "subscribe">> {
  setItem: (item: E | E[]) => void;
  deleteItem: (key: K | K[]) => void;
  hasItem: (key: K) => boolean;
  getItem: GetItem<K, E>;
}

export interface UseMapDataProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> {
  initialData?: E[];
}

export const useMapData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({ initialData }: UseMapDataProps<K, E> = {}): UseMapDataReturn<K, E> => {
  const dataRef = useInitializerRef(() => {
    const data = new Map<K, E>();
    if (initialData == null) return data;
    initialData.forEach((i) => data.set(i.key, i));
    return data;
  });
  const listenersRef = useInitializerRef(() => new Map<() => void, K>());
  const notifyListeners = useCallback((keys: K[]) => {
    const keysSet = new Set(keys);
    listenersRef.current.forEach((key, listener) => {
      if (keysSet.has(key)) listener();
    });
  }, []);
  const setItem = useCallback((item: E | E[]) => {
    const items = array.toArray(item);
    const itemKeys = items.map((i) => i.key);
    items.forEach((i) => dataRef.current.set(i.key, i));
    notifyListeners(itemKeys);
  }, []);
  const deleteItem = useCallback((key: K | K[]) => {
    const keys = array.toArray(key);
    keys.forEach((k) => dataRef.current.delete(k));
    notifyListeners(keys);
  }, []);
  const subscribe = useCallback(
    (listener: () => void, key?: K) => {
      if (key == null) return () => {};
      listenersRef.current.set(listener, key);
      return () => {
        listenersRef.current.delete(listener);
      };
    },
    [listenersRef],
  );
  const getItem = useCallback(
    ((key?: K | K[]) => {
      if (key == null) return;
      if (Array.isArray(key))
        return key.map((k) => dataRef.current.get(k)).filter((i) => i != null);
      return dataRef.current.get(key);
    }) as GetItem<K, E>,
    [],
  );
  const hasItem = useCallback((key: K) => dataRef.current.has(key), []);
  return useMemo(
    () => ({ setItem, deleteItem, subscribe, getItem, hasItem }),
    [setItem, deleteItem, subscribe, getItem, hasItem],
  );
};

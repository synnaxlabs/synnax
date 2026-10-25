// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type record } from "@synnaxlabs/x";

import { useInitializerRef } from "@/hooks";
import { type FrameProps } from "@/list/Frame";
import { Store } from "@/store";

interface GetItem<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> {
  (key?: K): E | undefined;
  (key: K[]): E[];
}

/** Return value for {@link useMapData}. */
export interface UseMapDataReturn<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> extends Required<Pick<FrameProps<K, E>, "subscribe">> {
  /** Adds or replaces items, re-rendering only the rows that carry their keys. */
  setItem: (item: E | E[]) => void;
  deleteItem: (key: K | K[]) => void;
  /** Reads the live entries, so render must not call it: subscribe instead. */
  hasItem: (key: K) => boolean;
  /**
   * Reads the live entries, so render must not call it. Read an entry in render through
   * `subscribe`, as `useItem` does.
   */
  getItem: GetItem<K, E>;
}

export interface UseMapDataProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> {
  initialData?: E[];
}

/**
 * Holds list entries in a ref-backed map and notifies only the rows whose keys change.
 * Use it when entries update often and a re-render of the whole list would be too
 * expensive. It owns the entries, not their order: pass `data` to the frame yourself.
 */
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
  const { notifyListeners, subscribe } = Store.useKeyedListeners<K>();
  const setItem = (item: E | E[]) => {
    const items = array.toArray(item);
    const itemKeys = items.map((i) => i.key);
    items.forEach((i) => dataRef.current.set(i.key, i));
    notifyListeners(itemKeys);
  };
  const deleteItem = (key: K | K[]) => {
    const keys = array.toArray(key);
    keys.forEach((k) => dataRef.current.delete(k));
    notifyListeners(keys);
  };
  const getItem = ((key?: K | K[]) => {
    if (key == null) return;
    if (Array.isArray(key))
      return key.map((k) => dataRef.current.get(k)).filter((i) => i != null);
    return dataRef.current.get(key);
  }) as GetItem<K, E>;
  const hasItem = (key: K) => dataRef.current.has(key);
  return { setItem, deleteItem, subscribe, getItem, hasItem };
};

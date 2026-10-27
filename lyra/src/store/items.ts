// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, type destructor, type record } from "@synnaxlabs/x";
import { useSyncExternalStore } from "react";

export interface UseItemsProps<K extends record.Key, E> {
  keys: K[];
  /** Calls the listener when the item with the given key changes. */
  subscribe: (listener: () => void, key: K) => destructor.Destructor;
  /** Returns the stored items for the given keys. */
  getItem: (keys: K[]) => E[];
}

const createStore = <K extends record.Key, E>({
  keys,
  subscribe,
  getItem,
}: UseItemsProps<K, E>) => {
  let snapshot: E[] = [];
  return {
    subscribe: (listener: () => void): destructor.Destructor => {
      const destructors = keys.map((key) => subscribe(listener, key));
      return () => destructors.forEach((d) => d());
    },
    getSnapshot: (): E[] => {
      const next = getItem(keys);
      if (!compare.arraysEqual(next, snapshot)) snapshot = next;
      return snapshot;
    },
  };
};

/**
 * Returns the items for the given keys and re-renders when any of them changes. A
 * list's `getItem` and `subscribe` fit as they are. The result keeps its identity while
 * every item keeps its own, so `getItem` must return the stored items, not copies.
 */
export const useItems = <K extends record.Key, E>({
  keys,
  subscribe,
  getItem,
}: UseItemsProps<K, E>): E[] => {
  const store = createStore({ keys, subscribe, getItem });
  return useSyncExternalStore(store.subscribe, store.getSnapshot);
};

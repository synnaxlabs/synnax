// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { useMemo, useSyncExternalStore } from "react";

export interface UseItemsParams<K, E> {
  keys: K[];
  /** Calls the handler when the item with the given key changes. */
  subscribe: (handler: () => void, key: K) => destructor.Destructor;
  /** Returns the stored items for the given keys. */
  getItems: (keys: K[]) => E[];
}

const createStore = <K, E>({ keys, subscribe, getItems }: UseItemsParams<K, E>) => {
  let snapshot: E[] = [];
  return {
    subscribe: (handler: () => void): destructor.Destructor => {
      const releases = keys.map((key) => subscribe(handler, key));
      return () => releases.forEach((release) => release());
    },
    getSnapshot: (): E[] => {
      const next = getItems(keys);
      if (
        next.length !== snapshot.length ||
        next.some((item, i) => item !== snapshot[i])
      )
        snapshot = next;
      return snapshot;
    },
  };
};

/**
 * Returns the items for the given keys and re-renders when any of them changes. The
 * result keeps its identity while every item keeps its own, so getItems must return
 * the stored items rather than copies.
 */
export const useItems = <K, E>({
  keys,
  subscribe,
  getItems,
}: UseItemsParams<K, E>): E[] => {
  const store = useMemo(
    () => createStore({ keys, subscribe, getItems }),
    [keys, subscribe, getItems],
  );
  return useSyncExternalStore(store.subscribe, store.getSnapshot);
};

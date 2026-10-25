// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type destructor, type record } from "@synnaxlabs/x";

import { useInitializerRef } from "@/hooks";

export interface UseKeyedListenersReturn<K extends record.Key> {
  /** Calls each listener subscribed to one of the keys or to every key, once. */
  notifyListeners: (keys: K | K[]) => void;
  /**
   * Subscribes the listener to the key, or to every key when key is omitted. A listener
   * may subscribe to many keys; each destructor removes one subscription.
   */
  subscribe: (listener: () => void, key?: K) => destructor.Destructor;
}

interface Listeners<K extends record.Key> {
  keyed: Map<K, Set<() => void>>;
  unkeyed: Set<() => void>;
}

export const useKeyedListeners = <
  K extends record.Key,
>(): UseKeyedListenersReturn<K> => {
  const listenersRef = useInitializerRef<Listeners<K>>(() => ({
    keyed: new Map(),
    unkeyed: new Set(),
  }));
  const notifyListeners = (keys: K | K[]) => {
    const { keyed, unkeyed } = listenersRef.current;
    const notified = new Set(unkeyed);
    array.toArray(keys).forEach((k) => keyed.get(k)?.forEach((l) => notified.add(l)));
    notified.forEach((l) => l());
  };
  const subscribe = (listener: () => void, key?: K) => {
    const { keyed, unkeyed } = listenersRef.current;
    if (key == null) {
      unkeyed.add(listener);
      return () => {
        unkeyed.delete(listener);
      };
    }
    let listeners = keyed.get(key);
    if (listeners == null) {
      listeners = new Set();
      keyed.set(key, listeners);
    }
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
      if (listeners.size === 0 && keyed.get(key) === listeners) keyed.delete(key);
    };
  };
  return { notifyListeners, subscribe };
};

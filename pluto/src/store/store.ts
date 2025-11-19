// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { useInitializerRef } from "@/hooks";

export interface UseKeyedListenersReturn<K extends record.Key> {
  notifyListeners: (keys: K[]) => void;
  subscribe: (listener: () => void, key?: K) => destructor.Destructor;
}

export const useKeyedListeners = <
  K extends record.Key,
>(): UseKeyedListenersReturn<K> => {
  const listenersRef = useInitializerRef(() => new Map<() => void, K | undefined>());
  const notifyListeners = useCallback((keys: K[]) => {
    listenersRef.current.forEach((key, listener) => {
      if (key == null || keys.includes(key)) listener();
    });
  }, []);
  const subscribe = useCallback((listener: () => void, key?: K) => {
    if (listenersRef.current.has(listener)) {
      const prevKey = listenersRef.current.get(listener);
      console.warn(
        `[store] attempted to subscribe listener with key ${prevKey} to key ${key} without being unsubscribed first`,
      );
    }
    listenersRef.current.set(listener, key);
    return () => listenersRef.current.delete(listener);
  }, []);
  return { notifyListeners, subscribe };
};

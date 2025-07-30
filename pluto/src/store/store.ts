import { type Destructor, type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { useInitializerRef } from "@/hooks";

export interface UseKeyedListenersReturn<K extends record.Key> {
  notifyListeners: (keys: K[]) => void;
  subscribe: (listener: () => void, key?: K) => Destructor;
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
    listenersRef.current.set(listener, key);
    return () => listenersRef.current.delete(listener);
  }, []);
  return { notifyListeners, subscribe };
};

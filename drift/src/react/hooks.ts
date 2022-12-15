import { EffectCallback, useEffect, useRef } from "react";

import { useSelectWindowState } from "./selectors";

/**
 * A hook that allows a user to tap into the lifecycle of a window.
 * Maintains a similar API to useEffect. Executes the callback when the
 * window state changes to 'created', and cleans up when the window state
 * changes to 'closing'.
 *
 * @param cb - The callback to execute.
 * @param key - The key of the window to subscribe to.
 * If not provided, the current window is used.
 */
export const useWindowLifecycle = (cb: EffectCallback, key?: string): void => {
  const status = useSelectWindowState(key);
  const destructor = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (status === "created" && destructor.current == null) {
      const c = cb();
      if (c != null) destructor.current = c;
    }
    if (status === "closing" && destructor.current != null) {
      destructor.current();
      destructor.current = null;
    }
  }, [status, destructor]);
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { type EffectCallback, useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { useSelectWindow } from "@/react/selectors";
import { completeProcess, registerProcess } from "@/state";

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
  const win = useSelectWindow(key);
  const dispatch = useDispatch();
  const destructor = useRef<(() => void) | null>(null);
  useEffect(() => {
    if (win == null) return;
    const { stage } = win;
    if (stage === "created" && destructor.current == null) {
      const c = cb();
      if (c != null) destructor.current = c;
      dispatch(registerProcess({ key: win.key }));
    } else if (
      (stage === "closing" || stage === "reloading") &&
      destructor.current != null
    ) {
      destructor.current();
      destructor.current = null;
      dispatch(completeProcess({ key: win.key }));
    }
  }, [win]);
};

export const useAsyncWindowLifecycle = (
  cb: () => Promise<destructor.Async | undefined>,
  key?: string,
): void => {
  const win = useSelectWindow(key);
  const dispatch = useDispatch();
  const destructor = useRef<destructor.Async | null>(null);
  const promiseOut = useRef<boolean>(false);
  useEffect(() => {
    if (win == null) return;
    const { stage } = win;
    if (stage === "created" && destructor.current == null) {
      promiseOut.current = true;
      cb()
        .then((d) => {
          destructor.current = d ?? (async () => {});
          dispatch(registerProcess({ key: win.key }));
        })
        .catch(console.error)
        .finally(() => {
          promiseOut.current = false;
        });
    } else if (
      (stage === "closing" || stage === "reloading") &&
      destructor.current != null
    ) {
      const f = destructor.current;
      destructor.current = null;
      f()
        .then(() => {
          destructor.current = null;
        })
        .catch(console.error)
        .finally(() => {
          dispatch(completeProcess({ key: win.key }));
        });
    }
  }, [win]);
};

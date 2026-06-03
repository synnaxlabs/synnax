// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, scheduler } from "@synnaxlabs/x";
import { type DependencyList, useEffect } from "react";

export type AsyncEffectCallback = (
  signal: AbortSignal,
) => Promise<void | destructor.Destructor | destructor.Async>;

/**
 * A React hook that runs an asynchronous effect with proper cleanup handling.
 *
 * This hook is similar to React's `useEffect` but designed for async operations.
 * It provides an AbortSignal for cancellation and handles both synchronous and
 * asynchronous cleanup functions.
 *
 * @param effect - The async effect function to run. Receives an AbortSignal for cancellation.
 * @param deps - Optional dependency array. The effect will re-run when dependencies change.
 *
 * @example
 * ```ts
 * useAsyncEffect(async (signal) => {
 *   const response = await fetch('/api/data', { signal });
 *   const data = await response.json();
 *   if (signal.aborted) return;
 *   setData(data);
 *
 *   return async () => {
 *     // Async cleanup
 *     await cleanup();
 *   };
 * }, []);
 * ```
 *
 * @example
 * ```ts
 * useAsyncEffect(async (signal) => {
 *   const subscription = observable.subscribe(data => {
 *     if (signal.aborted) return;
 *     setData(data);
 *   });
 *
 *   return () => {
 *     // Sync cleanup
 *     subscription.unsubscribe();
 *   };
 * }, [observable]);
 * ```
 */
export const useAsyncEffect = (
  effect: AsyncEffectCallback,
  deps?: DependencyList,
): void => {
  useEffect(() => {
    const controller = new AbortController();
    const signal = controller.signal;
    const effectFn = async () => {
      // flush the task queue so that the effect from the previous render has a chance
      // to finish before the cleanup function is called.
      await scheduler.flushTaskQueue();
      const maybeCleanup = await effect(signal);
      return maybeCleanup;
    };
    const cleanupPromise = effectFn().catch(console.error);
    return () => {
      controller.abort();
      cleanupPromise.then((cleanupFn) => cleanupFn?.()).catch(console.error);
    };
  }, deps);
};

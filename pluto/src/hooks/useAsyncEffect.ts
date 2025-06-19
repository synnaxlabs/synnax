// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AsyncDestructor, type Destructor, flushTaskQueue } from "@synnaxlabs/x";
import { type DependencyList, useEffect } from "react";

export type AsyncEffectCallback = (
  signal: AbortSignal,
) => Promise<void | Destructor | AsyncDestructor>;

export const useAsyncEffect = (
  effect: AsyncEffectCallback,
  deps?: DependencyList,
  onError: (error: unknown) => void = console.error,
): void => {
  useEffect(() => {
    const controller = new AbortController();
    const signal = controller.signal;
    const effectFn = async () => {
      // flush the task queue so that the effect from the previous render has a chance
      // to finish before the cleanup function is called.
      await flushTaskQueue();
      const maybeCleanup = await effect(signal);
      return maybeCleanup;
    };
    const cleanupPromise = effectFn().catch(onError);
    return () => {
      controller.abort();
      cleanupPromise.then((cleanupFn) => cleanupFn?.()).catch(onError);
    };
  }, deps);
};

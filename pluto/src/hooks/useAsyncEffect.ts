import { flushTaskQueue } from "@synnaxlabs/x";
import { type DependencyList, useEffect } from "react";

type AsyncEffectCallback = (signal: AbortSignal) => Promise<void | (() => void)>;

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

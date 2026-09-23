import { type destructor } from "@synnaxlabs/x";
import { type DependencyList } from "react";
/**
 * The body of a {@link useAsyncEffect}. The signal aborts when the effect re-runs or
 * the component unmounts; check it after every await before touching state. Return a
 * destructor, sync or async, to tear down what the effect set up.
 */
export type AsyncEffectCallback = (signal: AbortSignal) => Promise<void | destructor.Destructor | destructor.Async>;
/**
 * A React hook that runs an asynchronous effect with proper cleanup handling. This hook
 * is similar to React's `useEffect` but designed for async operations. It provides an
 * AbortSignal for cancellation and handles both synchronous and asynchronous cleanup
 * functions.
 * @param deps - Optional dependency array. The effect will re-run when dependencies
 * change.
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
export declare const useAsyncEffect: (effect: AsyncEffectCallback, deps?: DependencyList) => void;
//# sourceMappingURL=useAsyncEffect.d.ts.map
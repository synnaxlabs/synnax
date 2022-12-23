import { useEffect } from "react";

/* An async version of React.Destructor */
// eslint-disable-next-line @typescript-eslint/no-invalid-void-type
export type AsyncDestructor = Promise<(() => void) | void>;

/** An async version of React.EffectCallback */
export type AsyncEffectCallback = () => AsyncDestructor;

/**
 * A version of useEffect that supports async functions and destructors. NOTE: The behavior
 * of this hook hasn't been carefully though out, so it may produce unexpected results.
 * Caveat emptor.
 *
 * @param effect - The async effect callback.
 * @param deps - The dependencies of the effect.
 */
export const useAsyncEffect = (effect: AsyncEffectCallback, deps?: unknown[]): void => {
  useEffect(() => {
    const p = effect();
    return () => {
      p.then((d) => d?.()).catch((e) => console.error(e));
    };
  }, deps);
};

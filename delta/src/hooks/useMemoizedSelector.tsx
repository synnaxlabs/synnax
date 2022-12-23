import { useCallback } from "react";

import memoize from "proxy-memoize";
import { useSelector } from "react-redux";

/**
 * A memoized version of the redux useSelector hook. Only re-renders when the portions
 * of state accessed by the selector OR its dependencies change.
 *
 * @param selector - The selector function. NOTE: Avoid using object destructuring in the
 * selector, as it may cauase issues with memoization.
 * @param deps - The dependencies of the selector. If not provided, the selector will only
 * re-run when the state changes.
 * @returns The result of the selector.
 */
export const useMemoSelect = <S extends object, R>(
  selector: (state: S) => R,
  deps: unknown[] = []
): R => useSelector(useCallback(memoize(selector), deps));

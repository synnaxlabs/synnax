import { useCallback } from "react";

import type { KeyedRecord } from "@synnaxlabs/pluto";
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
  deps: unknown[]
): R => useSelector(useCallback(memoize(selector), deps));

export const selectByKeys = <S extends KeyedRecord<S>>(
  state: S[] | Record<string, S>,
  keys?: string[]
): S[] => {
  if (!Array.isArray(state)) state = Object.values(state);
  if (keys == null) return state;
  return state.filter((s) => keys.includes(s.key));
};

export const selectByKey = <S extends KeyedRecord<S>>(
  state: Record<string, S>,
  key?: string | null,
  defaultKey?: string | null
): S | null | undefined => {
  if (key == null) key = defaultKey;
  if (key == null) return null;
  return state[key];
};

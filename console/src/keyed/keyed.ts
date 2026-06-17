import { type PayloadAction } from "@reduxjs/toolkit";
import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { useMemoSelect } from "@/hooks";

export type Payload<K extends record.Key = string> = {
  key: K;
};

/**
 * Builds a memoized selector hook bound to a contextual key. The returned hook resolves
 * the key via useKey, then memo-selects against it, re-running only when the key or the
 * selected portion of state changes.
 *
 * @param selector - Selects a value from state for a given key.
 * @param useKey - Resolves the contextual key (e.g. the active layout key).
 */
export const createSelector =
  <S extends object, R, K extends record.Key = string>(
    selector: (state: S, key: K) => R,
    useKey: () => K,
  ): (() => R) =>
  () => {
    const key = useKey();
    return useMemoSelect((state: S) => selector(state, key), [key]);
  };

/**
 * Builds a record of memoized selector hooks in one call. Each entry maps the name of a
 * hook to a (state, key) selector; every hook resolves its contextual key via the shared
 * useKey. Equivalent to calling createSelector once per entry.
 *
 * @param useKey - Resolves the contextual key shared by every selector.
 * @param selectors - Maps each hook name to its (state, key) selector.
 */
export const createSelectors = <
  K extends record.Key,
  Sels extends Record<string, (state: never, key: K) => unknown>,
>(
  useKey: () => K,
  selectors: Sels,
): { [N in keyof Sels]: () => ReturnType<Sels[N]> } => {
  const hooks: Record<string, () => unknown> = {};
  for (const name in selectors) hooks[name] = createSelector(selectors[name], useKey);
  return hooks as { [N in keyof Sels]: () => ReturnType<Sels[N]> };
};

export const createDispatchHandler =
  <
    P extends Payload<K>,
    ObjectKey extends Exclude<keyof P, "key">,
    K extends record.Key = string,
  >(
    objectKey: ObjectKey,
    actionFactory: (payload: P) => PayloadAction<P>,
    useKey: () => K,
  ): (() => (value: P[ObjectKey]) => void) =>
  () => {
    const dispatch = useDispatch();
    const key = useKey();
    return useCallback(
      (value: P[ObjectKey]) =>
        dispatch(actionFactory({ key, [objectKey]: value } as P)),
      [dispatch, actionFactory, key, objectKey],
    );
  };

/**
 * A single dispatch-handler spec: the payload field the returned handler sets, paired
 * with the action that sets it.
 */
export type DispatchHandlerSpec<K extends record.Key = string> = readonly [
  objectKey: PropertyKey,
  actionFactory: (payload: never) => PayloadAction<Payload<K>>,
];

/** The hook produced for a given {@link DispatchHandlerSpec}. */
export type DispatchHandlerHook<S extends DispatchHandlerSpec<record.Key>> =
  S extends readonly [PropertyKey, (payload: infer P) => PayloadAction<infer _>]
    ? () => (value: P[Exclude<keyof P, "key">]) => void
    : never;

/**
 * Builds a record of dispatch-handler hooks in one call. Each spec is a
 * [objectKey, actionFactory] tuple keyed by the name of the hook to produce; every hook
 * resolves its contextual key via the shared useKey. Equivalent to calling
 * createDispatchHandler once per entry.
 *
 * @param useKey - Resolves the contextual key shared by every handler.
 * @param specs - Maps each hook name to its [objectKey, actionFactory] spec.
 */
export const createDispatchHandlers = <
  K extends record.Key,
  const S extends Record<string, DispatchHandlerSpec<K>>,
>(
  useKey: () => K,
  specs: S,
): { [N in keyof S]: DispatchHandlerHook<S[N]> } => {
  const handlers: Record<string, () => (value: never) => void> = {};
  for (const name in specs) {
    const [objectKey, actionFactory] = specs[name];
    handlers[name] = createDispatchHandler(
      objectKey as never,
      actionFactory as (payload: Payload<K>) => PayloadAction<Payload<K>>,
      useKey,
    );
  }
  return handlers as { [N in keyof S]: DispatchHandlerHook<S[N]> };
};

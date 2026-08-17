// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type primitive, state } from "@synnaxlabs/x";
import {
  type Ref,
  type RefCallback,
  type RefObject,
  useCallback,
  useRef,
  useState as reactUseState,
} from "react";

/**
 * A ref that satisfies the interface of useState, but returns a ref as the first
 * element of the tuple. This is useful when you want to keep a piece of state but don't
 * want its changes to trigger a re-render.
 *
 * @param initialValue - The initial value of the ref.
 * @returns a tuple containing the ref and the pseudo-setState function.
 */
export const useStateRef = <T extends state.State>(
  initialValue: state.Initial<T>,
): [RefObject<T>, state.Setter<T>] => {
  const ref = useRef<T>(state.executeInitialSetter(initialValue));
  const setValue: state.Setter<T> = useCallback((setter) => {
    ref.current = state.executeSetter(setter, ref.current);
  }, []);
  return [ref, setValue];
};

/**
 * Use synced ref keeps the provided value in sync with the returned ref. This is
 * useful when you want access to a piece of state but don't want it's changes
 * to trigger a re-render.
 *
 * @param value - The value to keep in sync with the ref.
 * @returns a ref that is kept in sync with the provided value.
 */
export const useSyncedRef = <T>(value: T): RefObject<T> => {
  const ref = useRef<T>(value);
  ref.current = value;
  return ref;
};

export const useInitializerRef = <T>(initializer: () => T): RefObject<T> => {
  const initializedRef = useRef<boolean>(false);
  const ref = useRef<T | null>(null);
  if (!initializedRef.current) {
    ref.current = initializer();
    initializedRef.current = true;
  }
  return ref as RefObject<T>;
};

/**
 * Combines multiple refs into one. Note that the returned ref callback will not be
 * updated when the provided refs changes. These refs are only set once, and are assumed
 * to be static throughout the lifetime of the component.
 *
 * @param refs - The refs to combine.
 * @returns - A callback ref that will set all of the provided refs.
 */
export const useCombinedRefs = <T>(
  ...refs: Array<Ref<T> | null | undefined>
): RefCallback<T> =>
  useCallback(
    (el) =>
      refs.forEach((r) => {
        if (r == null) return;
        if (typeof r === "function") r(el);
        else r.current = el;
      }, el),
    [],
  );

/**
 * Keeps a piece of state and a ref to it. The ref is assigned by the setter itself, so
 * it is current as soon as the setter returns, before the re-render it triggers.
 *
 * @param initialState - The initial state, or a function that lazily computes it.
 * @returns a tuple of the state, its setter, and a ref holding the latest value.
 */
export const useCombinedStateAndRef = <T extends primitive.Value | object>(
  initialState: state.Initial<T>,
): [T, state.Setter<T>, RefObject<T>] => {
  const ref = useInitializerRef<T>(() => state.executeInitialSetter<T>(initialState));
  const [s, setS] = reactUseState<T>(() => ref.current);

  const setStateAndRef: state.Setter<T> = useCallback((nextState): void => {
    ref.current = state.executeSetter<T>(nextState, ref.current);
    setS(ref.current);
  }, []);

  return [s, setStateAndRef, ref];
};

export const usePrevious = <T>(value: T): T | undefined => {
  const ref = useRef<T>(undefined);
  const prev = ref.current;
  ref.current = value;
  return prev;
};

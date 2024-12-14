// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Primitive } from "@synnaxlabs/x";
import {
  type ForwardedRef,
  type MutableRefObject,
  type RefCallback,
  useCallback,
  useRef,
  useState as reactUseState,
} from "react";

import { state } from "@/state";

/**
 * A ref that satisfies the interface of useState, but returns a ref as the first
 * element of the tuple. This is useful when you want to keep a piece of state
 * but don't want it's changes to trigger a re-render.
 *
 * @param initialValue - The initial value of the ref.
 * @returns a tuple containing the ref and the pseudo-setState function.
 */
export const useStateRef = <T extends state.State>(
  initialValue: state.Initial<T>,
): [MutableRefObject<T>, state.Set<T>] => {
  const ref = useRef<T>(state.executeInitialSetter(initialValue));
  const setValue: state.Set<T> = useCallback((setter) => {
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
export const useSyncedRef = <T>(value: T): MutableRefObject<T> => {
  const ref = useRef<T>(value);
  ref.current = value;
  return ref;
};

/**
 * Combines multiple refs into one. Note that the returned ref callback will not be
 * updated when the provided refs changes. These refs are only set once, and are assumed
 * to be static throughout the lifetime of the component.
 *
 * @param refs - The refs to combine.
 * @returns - A callback ref that will set all of the provided refs.
 */
export const useCombinedRefs = <T>(...refs: Array<ForwardedRef<T>>): RefCallback<T> =>
  useCallback(
    (el) =>
      refs.forEach((r) => {
        if (r == null) return;
        if (typeof r === "function") r(el);
        else r.current = el;
      }, el),
    [],
  );

export const useCombinedStateAndRef = <T extends Primitive | object>(
  initialState: state.Initial<T>,
): [T, state.Set<T>, React.MutableRefObject<T>] => {
  const ref = useRef<T | null>(null);
  const [s, setS] = reactUseState<T>(() => {
    const s = state.executeInitialSetter<T>(initialState);
    ref.current = s;
    return s;
  });

  const setStateAndRef: state.Set<T> = useCallback(
    (nextState): void => {
      setS((p) => {
        ref.current = state.executeSetter<T>(nextState, p);
        return ref.current;
      });
    },
    [setS],
  );

  return [s, setStateAndRef, ref as React.MutableRefObject<T>];
};

export const usePrevious = <T>(value: T): T | undefined => {
  const ref = useRef<T>(undefined);
  const prev = ref.current;
  ref.current = value;
  return prev;
};

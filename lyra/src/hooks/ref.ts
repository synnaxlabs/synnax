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
  useInsertionEffect,
  useRef,
  useState,
} from "react";

/**
 * A ref that satisfies the interface of useState, but returns a ref as the first
 * element of the tuple. This is useful when you want to keep a piece of state but don't
 * want its changes to trigger a re-render.
 * @returns a tuple containing the ref and the pseudo-setState function.
 */
export const useStateRef = <T extends state.State>(
  initialValue: state.Initial<T>,
): [RefObject<T>, state.Setter<T>] => {
  const ref = useRef<T>(state.executeInitialSetter(initialValue));
  const setValue: state.Setter<T> = (setter) => {
    ref.current = state.executeSetter(setter, ref.current);
  };
  return [ref, setValue];
};

/**
 * Keeps the returned ref in sync with the provided value. Use it to read the latest
 * value from an effect or a handler without re-running or re-creating it. The ref
 * updates when a render commits, before any effect runs, so never read it in render.
 * @returns a ref that is kept in sync with the provided value.
 */
export const useSyncedRef = <T>(value: T): RefObject<T> => {
  const ref = useRef<T>(value);
  useInsertionEffect(() => {
    ref.current = value;
  });
  return ref;
};

/**
 * Holds a ref whose value is built on the first render and kept for the component's
 * life. Use it in place of `useRef(expensive())`, which builds a fresh value on every
 * render and throws it away. The value cannot be null or undefined, because the hook
 * builds it again while it is either.
 *
 * @example const store = useInitializerRef(() => new Store());
 */
export const useInitializerRef = <T extends {}>(initializer: () => T): RefObject<T> => {
  const ref = useRef<T | null>(null);
  if (ref.current == null) ref.current = initializer();
  return ref as RefObject<T>;
};

/**
 * Combines multiple refs into one. The returned callback keeps one identity, so a ref
 * that changes after mount takes effect only when the element mounts again. A callback
 * ref may return a cleanup, which runs when the element detaches.
 *
 * @returns - A callback ref that will set all of the provided refs.
 */
export const useCombinedRefs = <T>(
  ...refs: Array<Ref<T> | null | undefined>
): RefCallback<T> => {
  const refsRef = useSyncedRef(refs);
  return (el) => {
    const cleanups = refsRef.current.map((r) => {
      if (r == null) return null;
      if (typeof r !== "function") {
        r.current = el;
        return () => {
          r.current = null;
        };
      }
      const cleanup = r(el);
      return typeof cleanup === "function" ? cleanup : () => r(null);
    });
    return () => cleanups.forEach((cleanup) => cleanup?.());
  };
};

/**
 * Keeps a piece of state and a ref to it. The ref is assigned by the setter itself, so
 * it is current as soon as the setter returns, before the re-render it triggers.
 * @param initialState - The initial state, or a function that lazily computes it.
 * @returns a tuple of the state, its setter, and a ref holding the latest value.
 */
export const useCombinedStateAndRef = <T extends primitive.Value | object>(
  initialState: state.Initial<T>,
): [T, state.Setter<T>, RefObject<T>] => {
  const [s, setS] = useState<T>(() => state.executeInitialSetter<T>(initialState));
  const ref = useRef<T>(s);

  const setStateAndRef: state.Setter<T> = (nextState): void => {
    ref.current = state.executeSetter<T>(nextState, ref.current);
    setS(ref.current);
  };

  return [s, setStateAndRef, ref];
};

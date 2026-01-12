// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, debounce as debounceF, type direction } from "@synnaxlabs/x";
import { type RefCallback, useCallback, useEffect, useRef } from "react";

import { useSyncedRef } from "@/hooks/ref";
import { compareArrayDeps, useMemoCompare } from "@/memo";

export interface UseResizeOpts {
  /**
   * A list of triggers that should cause the callback to be called.
   */
  triggers?: direction.Direction[];
  /**  Debounce the resize event by this many milliseconds.
  Useful for preventing expensive renders until resizing has stopped. */
  debounce?: number;
  /** If false, the hook wont observe the element. Defaults to true. */
  enabled?: boolean;
}

export type UseResizeHandler = <E extends HTMLElement>(box: box.Box, el: E) => void;

/**
 * Tracks the dimensions of an element and executes a callback when they change.
 *
 * @param onResize - A callback that receives a box representing the dimensions and
 * position of the element.
 * @param opts -  Options for the hook. See UseResizeOpts.
 *
 * @returns a ref callback to attach to the desire element.
 */
export const useResize = <E extends HTMLElement>(
  onResize: UseResizeHandler,
  opts: UseResizeOpts = {},
): RefCallback<E> => {
  const { triggers = [], debounce = 0, enabled = true } = opts;
  const prev = useRef<box.Box>(box.ZERO);
  const ref = useRef<E | null>(null);
  const obs = useRef<ResizeObserver | null>(null);
  const memoTriggers = useMemoCompare(() => triggers, compareArrayDeps, [
    triggers,
  ] as const);

  const startObserving = useCallback(
    (el: HTMLElement) => {
      if (obs.current != null) obs.current.disconnect();
      prev.current ??= box.ZERO;
      const deb = debounceF(() => {
        const next = box.construct(el);
        if (shouldResize(memoTriggers, prev.current, next)) {
          prev.current = next;
          onResize(next, ref.current as E);
        }
      }, debounce);
      obs.current = new ResizeObserver(deb);
      obs.current.observe(el);
    },
    [memoTriggers, onResize, debounce],
  );

  useEffect(() => {
    if (ref.current != null && enabled) startObserving(ref.current);
    return () => obs.current?.disconnect();
  }, [startObserving, enabled]);

  return useCallback(
    (el: E | null) => {
      ref.current = el;
      if (el != null && enabled) startObserving(el);
    },
    [startObserving],
  );
};

const shouldResize = (
  triggers: direction.Direction[],
  prev: box.Box,
  next: box.Box,
): boolean => {
  if (triggers.length === 0) return !box.equals(next, prev);
  if (triggers.includes("x") && box.width(prev) !== box.width(next)) return true;
  if (triggers.includes("y") && box.height(prev) !== box.height(next)) return true;
  return false;
};

export const useWindowResize = (onResize: UseResizeHandler) => {
  const onResizeRef = useSyncedRef(onResize);
  useEffect(() => {
    const handler = () =>
      onResizeRef.current(
        box.construct(window.innerWidth, window.innerHeight),
        window.document.documentElement,
      );
    window.addEventListener("resize", handler);
    return () => window.removeEventListener("resize", handler);
  }, []);
};

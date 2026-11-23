// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  box,
  type CrudeTimeSpan,
  type direction,
  scheduler,
  TimeSpan,
} from "@synnaxlabs/x";
import { type RefCallback, useEffect, useRef } from "react";

import { useSyncedRef } from "@/hooks/ref";

/** Options for {@link useResize}. */
export interface UseResizeOpts {
  /** A list of triggers that should cause the callback to be called. */
  triggers?: direction.Direction[];
  /** Debounce the resize event by this duration. Useful for preventing expensive
  renders until resizing has stopped. */
  debounce?: CrudeTimeSpan;
  /** If false, the hook wont observe the element. Defaults to true. */
  enabled?: boolean;
}

/**
 * Called with the element's new bounding box and the element itself. Fires on the
 * observer's callback, so it runs outside React's render pass.
 */
export type UseResizeHandler = <E extends HTMLElement>(box: box.Box, el: E) => void;

/**
 * Tracks the dimensions of an element and executes a callback when they change.
 * @param onResize - A callback that receives a box representing the dimensions and
 * position of the element.
 * @param opts - Options for the hook. See UseResizeOpts.
 * @returns a ref callback to attach to the desire element.
 */
export const useResize = <E extends HTMLElement>(
  onResize: UseResizeHandler,
  opts: UseResizeOpts = {},
): RefCallback<E> => {
  const { triggers = [], debounce = 0, enabled = true } = opts;
  const debouncePeriod = new TimeSpan(debounce).valueOf();
  const onResizeRef = useSyncedRef(onResize);
  const triggersRef = useSyncedRef(triggers);
  const prev = useRef<box.Box>(box.ZERO);
  const ref = useRef<E | null>(null);
  const observed = useRef<E | null>(null);
  const observation = useRef<{
    el: E;
    debouncePeriod: bigint;
    observer: ResizeObserver;
    report: scheduler.DebouncedFn<[]>;
  } | null>(null);
  const stop = (): void => {
    observation.current?.observer.disconnect();
    observation.current?.report.cancel();
    observation.current = null;
  };

  // React calls this after each render. Each new observation reports the box again, so
  // this observes only when the element or options change.
  const watch = (el: E | null): void => {
    const target = enabled ? el : null;
    const current = observation.current;
    if (current?.el === target && current.debouncePeriod === debouncePeriod) return;
    stop();
    if (target == null) return;
    // A new element reports its first box, even one that the last element had.
    if (target !== observed.current) prev.current = box.ZERO;
    observed.current = target;
    const report = scheduler.debounce(() => {
      const next = box.construct(target);
      if (!shouldResize(triggersRef.current, prev.current, next)) return;
      prev.current = next;
      onResizeRef.current(next, target);
    }, debouncePeriod);
    const observer = new ResizeObserver(() => report());
    observer.observe(target);
    observation.current = { el: target, debouncePeriod, observer, report };
  };

  useEffect(() => watch(ref.current), [watch]);
  useEffect(() => stop, []);

  return (el: E | null) => {
    ref.current = el;
    if (el != null) watch(el);
  };
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

/** Options for {@link useWindowResize}. */
export interface UseWindowResizeOptions {
  /** If false, the hook stops listening. Defaults to true. */
  enabled?: boolean;
}

/**
 * Calls onResize with the viewport's box whenever the window resizes. The handler is
 * read through a ref, so a fresh closure each render costs nothing.
 */
export const useWindowResize = (
  onResize: UseResizeHandler,
  opts: UseWindowResizeOptions = {},
) => {
  const { enabled = true } = opts;
  const onResizeRef = useSyncedRef(onResize);
  useEffect(() => {
    if (!enabled) return;
    const handler = () =>
      onResizeRef.current(
        box.construct(window.innerWidth, window.innerHeight),
        window.document.documentElement,
      );
    window.addEventListener("resize", handler);
    return () => window.removeEventListener("resize", handler);
  }, [enabled]);
};

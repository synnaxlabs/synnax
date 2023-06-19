// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RefCallback, useCallback, useEffect, useRef, useState } from "react";

import { debounce as debounceF, Box, Direction, LooseDirectionT } from "@synnaxlabs/x";

import { compareArrayDeps, useMemoCompare } from "@/core/hooks";

/** A list of events that can trigger a resize. */
export type DirectionTrigger = "moveX" | "moveY" | "resizeX" | "resizeY";

export interface UseResizeOpts {
  /** A list of triggers that should cause the callback to be called. */
  triggers?: Array<DirectionTrigger | Direction>;
  /**  Debounce the resize event by this many milliseconds.
  Useful for preventing expensive renders until rezizing has stopped. */
  debounce?: number;
}

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
  onResize: (box: Box, el: E) => void,
  { triggers: _triggers = [], debounce = 0 }: UseResizeOpts
): RefCallback<E> => {
  const prev = useRef<Box>(Box.ZERO);
  const ref = useRef<E | null>(null);
  const obs = useRef<ResizeObserver | null>(null);
  const triggers = useMemoCompare(
    () => normalizeTriggers(_triggers),
    compareArrayDeps,
    [_triggers] as const
  );

  const startObserving = useCallback(
    (el: HTMLElement) => {
      if (obs.current != null) obs.current.disconnect();
      if (prev.current == null) prev.current = Box.ZERO;
      const deb = debounceF(() => {
        const next = new Box(el);
        if (shouldResize(triggers, prev.current, next)) {
          prev.current = next;
          onResize(next, ref.current as E);
        }
      }, debounce);
      obs.current = new ResizeObserver(deb);
      obs.current.observe(el);
    },
    [triggers, onResize, debounce]
  );

  useEffect(() => {
    if (ref.current != null) startObserving(ref.current);
    return () => obs.current?.disconnect();
  }, [startObserving]);

  return useCallback(
    (el: E | null) => {
      ref.current = el;
      if (el != null) startObserving(el);
    },
    [startObserving]
  );
};

export type UseSizeOpts = UseResizeOpts;

/**
 * Tracks the size of an element and returns it.
 *
 * @param opts - Options for the hook. See UseSizeOpts.
 *
 * @returns A Box representing the size of the element and a ref callback to attach to
 * the element.
 */
export const useSize = <E extends HTMLElement>(
  opts: UseSizeOpts
): [Box, RefCallback<E>] => {
  const [size, onResize] = useState<Box>(Box.ZERO);
  const ref = useResize<E>(onResize, opts);
  return [size, ref];
};

const normalizeTriggers = (
  triggers: Array<Direction | DirectionTrigger>
): DirectionTrigger[] =>
  triggers
    .map((t): DirectionTrigger | DirectionTrigger[] => {
      if (Direction.isValid(t))
        return new Direction(t as LooseDirectionT).equals("x")
          ? ["moveX", "resizeX"]
          : ["moveY", "resizeY"];
      return t as DirectionTrigger;
    })
    .flat();

const shouldResize = (
  triggers: Array<DirectionTrigger | Direction>,
  prev: Box,
  next: Box
): boolean => {
  if (triggers.length === 0) return !next.equals(prev);
  if (triggers.includes("resizeX") && prev.width !== next.width) return true;
  if (triggers.includes("resizeY") && prev.height !== next.height) return true;
  if (triggers.includes("moveX") && prev.left !== next.left) return true;
  if (triggers.includes("moveY") && prev.top !== next.top) return true;
  return false;
};

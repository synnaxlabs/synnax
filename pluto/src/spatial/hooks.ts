// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RefObject, useCallback, useLayoutEffect, useRef, useState } from "react";

import { ResizeObserver } from "@juggle/resize-observer";

import { Box, CSSBox, ZERO_BOX } from "./box";
import { Direction, isDirection } from "./core";

import { useDebouncedCallback } from "@/util/debounce";

interface BaseSizeProps {
  ref: RefObject<HTMLElement>;
  /**  Debounce the resize event by this many milliseconds.
  Useful for preventing expensive renders until rezizing has stopped. */
  debounce?: number;
}

export type Trigger = "moveX" | "moveY" | "resizeX" | "resizeY";

export const normalizeTriggers = (triggers: Array<Direction | Trigger>): Trigger[] =>
  triggers
    .map((t): Trigger | Trigger[] => {
      if (isDirection(t))
        return t === "horizontal" ? ["moveX", "resizeX"] : ["moveY", "resizeY"];
      return t as Trigger;
    })
    .flat();

export interface UseResizeProps extends BaseSizeProps {
  /** Called when the size of the element changes. */
  onResize: (box: Box) => void;
  triggers?: Array<Trigger | Direction>;
}

const useShouldResize = (
  triggers?: Array<Trigger | Direction>
): ((prev: Box, next: Box) => boolean) =>
  useCallback(
    (prev: Box, next: Box) => {
      if (triggers == null)
        return (
          prev.width !== next.width ||
          prev.height !== next.height ||
          prev.left !== next.left ||
          prev.top !== next.top
        );
      triggers = normalizeTriggers(triggers);
      if (prev.width !== next.width && triggers.includes("resizeX")) return true;
      if (prev.height !== next.height && triggers.includes("resizeY")) return true;
      if (prev.left !== next.left && triggers.includes("moveX")) return true;
      if (prev.top !== next.top && triggers.includes("moveY")) return true;
      return false;
    },
    [triggers?.length]
  );

/**
 *  useResize tracks the size of an element and calls a callback when it changes.
 * @param ref - A ref to the element to track.
 * @param opts -  Options for the hook. See useResizeOpts.
 * @returns The width and height of the element.
 */
export const useResize = ({
  ref,
  onResize,
  triggers,
  debounce = 0,
}: UseResizeProps): void => {
  const shouldResize = useShouldResize(triggers);

  const prev = useRef<Box>(ZERO_BOX);

  const debounced = useDebouncedCallback(
    (el: Element, p: Box, shouldResize: (prev: Box, next: Box) => boolean) => {
      const next = new CSSBox(el.getBoundingClientRect());
      if (shouldResize(p, next)) onResize(new CSSBox(el.getBoundingClientRect()));
      prev.current = next;
    },
    debounce,
    [onResize]
  );

  useLayoutEffect(() => {
    if (ref.current == null) return;
    debounced(ref.current, prev.current, shouldResize);
    const obs = new ResizeObserver(([{ target }]) =>
      debounced(target, prev.current, shouldResize)
    );
    obs.observe(ref.current);
    return () => obs.disconnect();
  }, [debounced, shouldResize]);
};

export type UseBoxProps = BaseSizeProps;

/**
 * useSize tracks the size of an element and returns it.
 * @param props - Props for the hook. See useSizeProps.
 * @returns The width and height of the element.
 */
export const useBox = (props: UseBoxProps): Box => {
  const [size, onResize] = useState<Box>(ZERO_BOX);
  useResize({ onResize, ...props });
  return size;
};

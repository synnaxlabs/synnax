// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RefObject, useLayoutEffect, useState } from "react";

import { ResizeObserver } from "@juggle/resize-observer";

import { debounce as debounceF } from "@/util/debounce";

interface BaseSizeProps {
  ref: RefObject<HTMLElement>;
  /**  Debounce the resize event by this many milliseconds.
  Useful for preventing expensive renders until rezizing has stopped. */
  debounce?: number;
}

export interface OnResizeProps {
  width: number;
  height: number;
  top: number;
  left: number;
}

export interface UseResizeProps extends BaseSizeProps {
  /** Called when the size of the element changes. */
  onResize: (size: OnResizeProps) => void;
}

/**
 *  useResize tracks the size of an element and calls a callback when it changes.
 * @param ref - A ref to the element to track.
 * @param opts -  Options for the hook. See useResizeOpts.
 * @returns The width and height of the element.
 */
export const useResize = ({ ref, onResize, debounce = 0 }: UseResizeProps): void => {
  useLayoutEffect(() => {
    const el = ref.current;
    if (el == null) return;
    const f = debounceF<(el: Element) => void>((el: Element) => {
      const { width, height, top, left } = el.getBoundingClientRect();
      onResize({ width, height, top, left });
    }, debounce);
    f(el);
    const resizeObserver = new ResizeObserver(([entry]) => f(entry.target));
    resizeObserver.observe(el);
    return () => resizeObserver.disconnect();
  }, [onResize, ref]);
};

export type UseSizeProps = BaseSizeProps;

export type UseSizeReturn = OnResizeProps;

/**
 * useSize tracks the size of an element and returns it.
 * @param props - Props for the hook. See useSizeProps.
 * @returns The width and height of the element.
 */
export const useSize = (props: UseSizeProps): UseSizeReturn => {
  const [size, onResize] = useState<OnResizeProps>({
    width: 0,
    height: 0,
    top: 0,
    left: 0,
  });
  useResize({ onResize, ...props });
  return size;
};

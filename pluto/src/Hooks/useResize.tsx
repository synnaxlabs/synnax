import { RefObject, useLayoutEffect, useState } from "react";
import { ResizeObserver } from "@juggle/resize-observer";
import { debounce as debounceF } from "@/util";

interface BaseSizeProps {
  ref: RefObject<HTMLElement>;
  /**  Debounce the resize event by this many milliseconds. 
  Useful for preventing expensive renders until rezizing has stopped. */
  debounce?: number;
}

export interface UseResizeProps extends BaseSizeProps {
  /** Called when the size of the element changes. */
  onResize: (size: { width: number; height: number }) => void;
}

/**
 *  useResize tracks the size of an element and calls a callback when it changes.
 * @param ref - A ref to the element to track.
 * @param opts -  Options for the hook. See useResizeOpts.
 * @returns The width and height of the element.
 */
export const useResize = ({ ref, onResize, debounce = 0 }: UseResizeProps) => {
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const f = debounceF((el: Element) => {
      const { width, height } = el.getBoundingClientRect();
      onResize({ width, height });
    }, debounce);
    f(el);
    const resizeObserver = new ResizeObserver(([entry]) => f(entry.target));
    resizeObserver.observe(el);
    return () => resizeObserver.disconnect();
  }, [onResize, ref]);
};

export interface UseSizeProps extends BaseSizeProps {}

/**
 * useSize tracks the size of an element and returns it.
 * @param props - Props for the hook. See useSizeProps.
 * @returns The width and height of the element.
 */
export const useSize = (
  props: UseSizeProps
): { width: number; height: number } => {
  const [size, onResize] = useState({ width: 0, height: 0 });
  useResize({ onResize, ...props });
  return size;
};

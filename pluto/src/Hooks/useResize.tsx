import { useEffect, useLayoutEffect, useState } from "react";
import { ResizeObserver } from "@juggle/resize-observer";

export type useResizeOpts = {
  /**  Debounce the resize event by this many milliseconds. 
  Useful for preventing expensive renders until rezizing has stopped. */
  debounce?: number;
};

/**
 *  useResize tracks the size of an element and returns the width and height
 *  whenever it changes.
 * @param ref - A ref to the element to track.
 * @param opts -  Options for the hook. See useResizeOpts.
 * @returns The width and height of the element.
 */
const useResize = (
  ref: React.RefObject<HTMLElement>,
  { debounce = 0 }: useResizeOpts = { debounce: 0 }
) => {
  const [size, setSize] = useState({ width: 0, height: 0 });
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    let timeoutF: NodeJS.Timeout;
    const resizeObserver = new ResizeObserver(([entry]) => {
      const f = () => {
        const { width, height } = entry.target.getBoundingClientRect();
        setSize({ width, height });
      };
      if (debounce != 0) {
        clearTimeout(timeoutF);
        timeoutF = setTimeout(f, debounce);
      } else {
        f();
      }
    });
    resizeObserver.observe(el);
    return () => {
      resizeObserver.disconnect();
    };
  }, [ref]);

  useEffect(() => {
    const el = ref.current;
    if (!el || size.width != 0 || size.height != 0) return;
    const { width, height } = el.getBoundingClientRect();
    setSize({ width, height });
  }, [ref]);

  return size;
};

export default useResize;

import { useEffect, useLayoutEffect, useState } from "react";
import { ResizeObserver } from "@juggle/resize-observer";
import debounceF from "../util/debounce";

export type useResizeOpts = {
  /**  Debounce the resize event by this many milliseconds. 
  Useful for preventing expensive renders until rezizing has stopped. */
  debounce?: number;
};

/** Checks if any element on the dom is currently being dragged */
const useDragging = (): boolean => {
  const [dragging, setDragging] = useState(false);
  useEffect(() => {
    const onMouseDown = () => setDragging(true);
    const onMouseUp = () => setDragging(false);
    document.addEventListener("mousedown", onMouseDown);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousedown", onMouseDown);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, []);
  return dragging;
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

  const dragging = useDragging();
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const f = debounceF((el: Element) => {
      const { width, height } = el.getBoundingClientRect();
      setSize({ width, height });
    }, debounce);
    f(el);
    const resizeObserver = new ResizeObserver(([entry]) => f(entry.target));
    resizeObserver.observe(el);
    return () => resizeObserver.disconnect();
  }, [ref]);

  return size;
};

export default useResize;

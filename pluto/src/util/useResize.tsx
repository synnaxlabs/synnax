import { useLayoutEffect, useState } from "react";

export type useResizeOpts = {
  debounce?: number;
};

export const useResize = (
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
  return size;
};

const useResizing = (): boolean => {
  const [resizing, setResizing] = useState(false);
  useLayoutEffect(() => {
    const handleResizeStart = () => setResizing(true);
    const handleResizeEnd = () => setResizing(false);
    window.addEventListener("resize", handleResizeStart);
    window.addEventListener("resize", handleResizeEnd);
    return () => {
      window.removeEventListener("resize", handleResizeStart);
      window.removeEventListener("resize", handleResizeEnd);
    };
  }, []);
  return resizing;
};

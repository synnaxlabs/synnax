import {
  CanvasHTMLAttributes,
  DetailedHTMLProps,
  ReactElement,
  useCallback,
  useMemo,
} from "react";

import { Box } from "@synnaxlabs/x";

import { useResize } from "../hooks";

import { VisContext } from "./Context";

type HTMLCanvasProps = DetailedHTMLProps<
  CanvasHTMLAttributes<HTMLCanvasElement>,
  HTMLCanvasElement
>;

export interface CanvasProps extends Omit<HTMLCanvasProps, "ref"> {
  worker: Worker;
}

export const Canvas = ({ worker }: CanvasProps): ReactElement => {
  const handleResize = useCallback(
    (box: Box, canvas: HTMLCanvasElement) => {
      const dpr = window.devicePixelRatio;
      const { clientWidth: cw, clientHeight: ch, width: w, height: h } = canvas;
      const needResize = w !== cw || h !== ch;
      if (needResize) [canvas.width, canvas.height] = [cw * dpr, ch * dpr];
      worker.postMessage({
        type: "canvas-resize",
        data: {
          box,
          dpr,
          viewport: needResize ? [canvas.width, canvas.height] : null,
        },
      });
    },
    [worker]
  );

  const handleSetProps = useCallback(
    (path: string, props: unknown) => {
      worker.postMessage({ type: "canvas-set-props", data: { path, props } });
    },
    [worker]
  );

  const value = useMemo(
    () => ({ parent: "", setProps: handleSetProps }),
    [handleSetProps]
  );

  const resizeRef = useResize(handleResize, { debounce: 100 });

  const refCallback = useCallback(
    (canvas: HTMLCanvasElement | null) => {
      if (canvas == null) return;
      resizeRef(canvas);
      const box = canvas.getBoundingClientRect();
      worker.postMessage({
        type: "canvas-bootstrap",
        data: { box, dpr: window.devicePixelRatio },
      });
    },
    [worker]
  );

  return (
    <VisContext.Provider value={value}>
      <canvas ref={refCallback} />
    </VisContext.Provider>
  );
};

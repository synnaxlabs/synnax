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

import { WorkerMessage } from "@/core/vis/worker";
import { useTypedWorker } from "@/worker/Context";

type HTMLCanvasProps = DetailedHTMLProps<
  CanvasHTMLAttributes<HTMLCanvasElement>,
  HTMLCanvasElement
>;

export interface CanvasProps extends Omit<HTMLCanvasProps, "ref"> {}

export const Canvas = ({ children }: CanvasProps): ReactElement => {
  const worker = useTypedWorker<WorkerMessage>("vis");
  const handleResize = useCallback(
    (box: Box, canvas: HTMLCanvasElement) => {
      const dpr = window.devicePixelRatio;
      const { clientWidth: cw, clientHeight: ch, width: w, height: h } = canvas;
      const needResize = w !== cw || h !== ch;
      if (needResize) [canvas.width, canvas.height] = [cw * dpr, ch * dpr];
      worker.send({
        type: "resize",
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
      worker.send({ type: "set-props", data: { path, props } });
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
      // transfer control of the canvas to the worker
      worker.postMessage({
        type: "bootstrap",
        data: { box, dpr: window.devicePixelRatio },
      });
    },
    [worker]
  );

  return (
    <VisContext.Provider value={value}>
      <canvas ref={refCallback} />
      {childre}
    </VisContext.Provider>
  );
};

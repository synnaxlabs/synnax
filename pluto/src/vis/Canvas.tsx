// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import {
  CanvasHTMLAttributes,
  DetailedHTMLProps,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

import { Box } from "@synnaxlabs/x";

import { Theming, useResize } from "..";

import { LineVisState } from "./line/core/line";

export interface VisCanvasContextValue {
  update: (d: any) => void;
}

type HTMLCanvasProps = DetailedHTMLProps<
  CanvasHTMLAttributes<HTMLCanvasElement>,
  HTMLCanvasElement
>;

export const CanvasContext = createContext<VisCanvasContextValue | null>(null);

export interface CanvasProps extends Omit<HTMLCanvasProps, "ref"> {}

export const useCanvas = (): VisCanvasContextValue => {
  const ctx = useContext(CanvasContext);
  if (ctx == null) {
    throw new Error("useCanvas must be used within a Canvas");
  }
  return ctx;
};

export const Canvas = ({ children, ...props }: CanvasProps): JSX.Element | null => {
  const [worker, setWorker] = useState<Worker | null>(null);
  const theme = Theming.useContext();

  const handleResize = useCallback(
    (box: Box, canvas: HTMLCanvasElement) => {
      const dpr = window.devicePixelRatio;
      const { clientWidth: cw, clientHeight: ch, width: w, height: h } = canvas;
      const needResize = w !== cw || h !== ch;
      if (needResize) [canvas.width, canvas.height] = [cw * dpr, ch * dpr];
      if (worker != null) {
        worker.postMessage({
          type: "resize",
          data: {
            box,
            dpr,
            viewport: needResize ? [canvas.width, canvas.height] : null,
          },
        });
      }
    },
    [worker]
  );

  const canvasRef = useResize(handleResize, {});

  useEffect(
    () => () => {
      if (worker != null) {
        worker.terminate();
      }
    },
    []
  );

  const refCallback = useCallback(
    (canvas: HTMLCanvasElement) => {
      if (worker != null || canvas == null) return;
      canvasRef(canvas);
      const w = new Worker(new URL("./worker.ts", import.meta.url));
      setWorker(w);
      const connParams = {
        host: "localhost",
        port: 9090,
        secure: false,
        username: "synnax",
        password: "seldon",
      };
      const offscreen = canvas.transferControlToOffscreen();
      const box = new Box(canvas);
      const dpr = window.devicePixelRatio;
      w.postMessage(
        {
          type: "bootstrap",
          data: { box, dpr, offscreen, connParams, theme: theme.theme },
        },
        [offscreen]
      );
    },
    [worker]
  );

  const handleUpdate = useCallback(
    (d: LineVisState) => {
      if (worker != null) {
        worker.postMessage({ type: "update", data: d });
      }
    },
    [worker]
  );

  return (
    <CanvasContext.Provider value={{ update: handleUpdate }}>
      <canvas {...props} ref={refCallback} />
      {worker != null && children}
    </CanvasContext.Provider>
  );
};

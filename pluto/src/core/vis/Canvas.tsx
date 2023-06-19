// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  CanvasHTMLAttributes,
  DetailedHTMLProps,
  ReactElement,
  useCallback,
  useRef,
} from "react";

import { Box } from "@synnaxlabs/x";

import { Aether } from "@/core/aether/main";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import { Canvas as WorkerCanvas, canvasState } from "@/core/vis/WorkerCanvas";

import "@/core/vis/Canvas.css";

type HTMLCanvasProps = DetailedHTMLProps<
  CanvasHTMLAttributes<HTMLCanvasElement>,
  HTMLCanvasElement
>;

export interface VisCanvasProps extends Omit<HTMLCanvasProps, "ref"> {}

const ZERO_PROPS = { region: Box.ZERO, dpr: 1 };

interface Canvases {
  gl: HTMLCanvasElement | null;
  canvas: HTMLCanvasElement | null;
  bootstrapped: boolean;
}

const bootstrapped = ({ gl, canvas }: Canvases): boolean =>
  canvas != null && gl != null;

export const VisCanvas = ({ children, ...props }: VisCanvasProps): ReactElement => {
  const {
    path,
    state: [, setState],
  } = Aether.use(WorkerCanvas.TYPE, ZERO_PROPS, canvasState);

  const canvases = useRef<Canvases>({ gl: null, canvas: null, bootstrapped: false });

  const handleResize = useCallback(
    (region: Box) =>
      bootstrapped(canvases.current) &&
      setState({ region, dpr: window.devicePixelRatio }),
    []
  );

  const resizeRef = useResize(handleResize, { debounce: 100 });

  const refCallback = useCallback((el: HTMLCanvasElement | null) => {
    resizeRef(el);
    if (el == null) return;
    if (el.className.includes("gl")) canvases.current.gl = el;
    else canvases.current.canvas = el;
    const { gl, canvas, bootstrapped } = canvases.current;
    if (gl == null || canvas == null || bootstrapped) return;
    canvases.current.bootstrapped = true;
    const glOffscreen = gl.transferControlToOffscreen();
    const canvasOffscreen = canvas.transferControlToOffscreen();
    const region = new Box(canvas.getBoundingClientRect());
    setState(
      {
        glCanvas: glOffscreen,
        canvasCanvas: canvasOffscreen,
        region,
        dpr: window.devicePixelRatio,
      },
      [glOffscreen, canvasOffscreen]
    );
  }, []);

  return (
    <>
      <canvas ref={refCallback} className={CSS.BM("canvas", "gl")} {...props} />
      <canvas ref={refCallback} className={CSS.BM("canvas", "2d")} {...props} />
      <Aether.Composite path={path}>
        {bootstrapped(canvases.current) && children}
      </Aether.Composite>
    </>
  );
};

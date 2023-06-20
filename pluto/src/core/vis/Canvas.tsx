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

export interface VisCanvasProps extends Omit<HTMLCanvasProps, "ref"> {
  resizeDebounce?: number;
}

const ZERO_PROPS = { region: Box.ZERO, dpr: 1 };

interface Canvases {
  gl: HTMLCanvasElement | null;
  lower2d: HTMLCanvasElement | null;
  upper2d: HTMLCanvasElement | null;
  bootstrapped: boolean;
}

const ZERO_CANVASES: Canvases = {
  gl: null,
  lower2d: null,
  upper2d: null,
  bootstrapped: false,
};

const bootstrapped = ({ gl, lower2d, upper2d }: Canvases): boolean =>
  [gl, lower2d, upper2d].every((c) => c != null);

export const VisCanvas = ({
  children,
  resizeDebounce: debounce = 100,
  ...props
}: VisCanvasProps): ReactElement => {
  const [{ path }, , setState] = Aether.use(WorkerCanvas.TYPE, canvasState, ZERO_PROPS);

  const canvases = useRef<Canvases>(ZERO_CANVASES);

  const handleResize = useCallback(
    (region: Box) =>
      bootstrapped(canvases.current) &&
      setState({ region, dpr: window.devicePixelRatio }),
    []
  );

  const resizeRef = useResize(handleResize, { debounce });

  const refCallback = useCallback((el: HTMLCanvasElement | null) => {
    resizeRef(el);
    if (el == null) return;

    // Store the canvas
    if (el.className.includes("gl")) canvases.current.gl = el;
    else if (el.className.includes("upper2d")) canvases.current.upper2d = el;
    else canvases.current.lower2d = el;
    const { gl, lower2d, upper2d, bootstrapped } = canvases.current;

    if (gl == null || lower2d == null || upper2d == null || bootstrapped) return;

    // Bootstrap the canvas
    canvases.current.bootstrapped = true;
    const glCanvas = gl.transferControlToOffscreen();
    const upper2dCanvas = upper2d.transferControlToOffscreen();
    const lower2dCanvas = lower2d.transferControlToOffscreen();
    setState(
      {
        glCanvas,
        upper2dCanvas,
        lower2dCanvas,
        region: new Box(gl),
        dpr: window.devicePixelRatio,
      },
      [glCanvas, upper2dCanvas, lower2dCanvas]
    );
  }, []);

  return (
    <>
      <canvas ref={refCallback} className={CSS.BM("canvas", "lower2d")} {...props} />
      <canvas ref={refCallback} className={CSS.BM("canvas", "gl")} {...props} />
      <canvas ref={refCallback} className={CSS.BM("canvas", "upper2d")} {...props} />
      <Aether.Composite path={path}>
        {bootstrapped(canvases.current) && children}
      </Aether.Composite>
    </>
  );
};

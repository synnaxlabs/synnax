// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/canvas/Canvas.css";

import { box, runtime, scale, xy } from "@synnaxlabs/x";
import {
  type CanvasHTMLAttributes,
  type DetailedHTMLProps,
  type ReactElement,
  type RefCallback,
  useCallback,
  useEffect,
  useRef,
} from "react";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import {
  useCombinedRefs,
  useResize,
  type UseResizeHandler,
  type UseResizeOpts,
} from "@/hooks";
import { canvas } from "@/vis/canvas/aether";

type HTMLDivProps = DetailedHTMLProps<
  CanvasHTMLAttributes<HTMLDivElement>,
  HTMLDivElement
>;

const ZERO_PROPS = { region: box.ZERO, dpr: 1, os: runtime.getOS() };

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

export interface CanvasProps extends Omit<HTMLDivProps, "ref"> {
  resizeDebounce?: number;
}

export const Canvas = ({
  children,
  resizeDebounce: debounce = 100,
  className,
  ...rest
}: CanvasProps): ReactElement => {
  const [{ path }, { bootstrapped, dpr }, setState] = Aether.use({
    type: canvas.Canvas.TYPE,
    schema: canvas.canvasStateZ,
    initialState: ZERO_PROPS,
  });

  const canvases = useRef<Canvases>({ ...ZERO_CANVASES });

  const initialResizeComplete = useRef(false);

  const handleResize = useCallback(
    (region: box.Box) => {
      if (canvases.current.bootstrapped) {
        setState(() => ({
          bootstrapped: true,
          region,
          dpr: window.devicePixelRatio,
          os: runtime.getOS({ default: "Windows" }),
        }));
        initialResizeComplete.current = true;
      }
    },
    [setState],
  );

  const elRef = useRef<HTMLDivElement | null>(null);
  const resizeRef = useResize(handleResize, { debounce });
  const combinedElRef = useCombinedRefs(elRef, resizeRef);

  useEffect(() => {
    // Handle device pixel ratio change i.e. when the user moves the window to a
    // different display.
    const handleChange = (): void => {
      if (
        window.devicePixelRatio === dpr ||
        !canvases.current.bootstrapped ||
        elRef.current == null
      )
        return;
      setState((p) => ({
        ...p,
        // We need to explicitly construct the region here because this callback
        // may race against the `useResize` callback and cause a stale region to
        // be used in state.
        region: box.construct(elRef.current ?? box.ZERO),
        dpr: window.devicePixelRatio,
      }));
    };
    window
      .matchMedia(`(resolution: ${window.devicePixelRatio}dppx)`)
      .addEventListener("change", handleChange, { once: true });
  }, [dpr]);

  // We want to trigger a re-render when the window is focused or blurred to ensure
  // that we wake up sleeping render contexts.
  useEffect(() => {
    const handler = () => {
      if (!canvases.current.bootstrapped || elRef.current == null) return;
      setState((p) => ({
        ...p,
        // We need to explicitly construct the region here because this callback
        // may race against the `useResize` callback and cause a stale region to
        // be used in state.
        region: box.construct(elRef.current ?? box.ZERO),
        glCanvas: undefined,
        upper2dCanvas: undefined,
        lower2dCanvas: undefined,
      }));
    };
    window.addEventListener("focus", handler);
    window.addEventListener("blur", handler);
  }, [setState]);

  const refCallback = useCallback(
    (el: HTMLCanvasElement | null) => {
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
          bootstrap: true,
          bootstrapped: false,
          region: box.construct(gl),
          dpr: window.devicePixelRatio,
          os: runtime.getOS({ default: "Windows" }),
        },
        [glCanvas, upper2dCanvas, lower2dCanvas],
      );
    },
    [setState],
  );

  return (
    <div
      ref={combinedElRef}
      className={CSS(CSS.B("canvas-container"), className)}
      {...rest}
    >
      <canvas
        ref={refCallback}
        className={CSS(CSS.B("canvas"), CSS.BM("canvas", "lower2d"))}
      />
      <canvas
        ref={refCallback}
        className={CSS(CSS.B("canvas"), CSS.BM("canvas", "gl"))}
      />
      <canvas
        ref={refCallback}
        className={CSS(CSS.B("canvas"), CSS.BM("canvas", "upper2d"))}
      />
      <Aether.Composite path={path}>{bootstrapped && children}</Aether.Composite>
    </div>
  );
};

export const useRegion = (
  handler: UseResizeHandler,
  opts?: UseResizeOpts,
): RefCallback<HTMLDivElement> =>
  useResize(
    useCallback(
      (b, el) => {
        const canvas = document.querySelector(".pluto-canvas--lower2d");
        if (canvas == null) return;
        handler(scale.XY.translate(xy.scale(box.topLeft(canvas), -1)).box(b), el);
      },
      [handler],
    ),
    opts,
  );

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
  useEffect,
  useState,
} from "react";

import { Box } from "@synnaxlabs/x";

import { Aether } from "@/core/aether/main";
import { useResize } from "@/core/hooks";
import {
  Bootstrap,
  Canvas as WorkerCanvas,
  CanvasState as WorkerCanvasProps,
} from "@/core/vis/WorkerCanvas";

type HTMLCanvasProps = DetailedHTMLProps<
  CanvasHTMLAttributes<HTMLCanvasElement>,
  HTMLCanvasElement
>;

export interface VisCanvasProps extends Omit<HTMLCanvasProps, "ref"> {}

export const VisCanvas = ({ children, ...props }: VisCanvasProps): ReactElement => {
  const bootstrap = Aether.useBootstrap<Bootstrap>();
  const {
    path,
    state: [, setCanvas],
  } = Aether.use<WorkerCanvasProps>(
    WorkerCanvas.TYPE,
    {
      region: Box.ZERO,
      dpr: 1,
    },
    "canvas-key"
  );

  const [glCanvas, setGlCanvas] = useState<HTMLCanvasElement | null>(null);
  const [canvasCanvas, setCanvasCanvas] = useState<HTMLCanvasElement | null>(null);
  const [bootstrapped, setBootstrapped] = useState(false);

  const handleResize = useCallback(
    (box: Box) => {
      if (!bootstrapped) return;
      const dpr = window.devicePixelRatio;
      setCanvas({
        region: box,
        dpr,
      });
    },
    [bootstrapped]
  );

  useEffect(() => {
    if (glCanvas == null || canvasCanvas == null) return;
    const glOffscreen = glCanvas.transferControlToOffscreen();
    const canvasOffscreen = canvasCanvas.transferControlToOffscreen();
    const region = new Box(glCanvas.getBoundingClientRect());
    bootstrap(
      {
        key: path[0],
        glCanvas: glOffscreen,
        canvasCanvas: canvasOffscreen,
        region,
        dpr: window.devicePixelRatio,
      },
      [glOffscreen, canvasOffscreen]
    );
    setBootstrapped(true);
  }, [glCanvas, canvasCanvas]);

  const resizeRef = useResize(handleResize, { debounce: 100 });

  const glRefCallback = useCallback((canvas: HTMLCanvasElement | null) => {
    resizeRef(canvas);
    if (glCanvas != null || canvas == null) return;
    setGlCanvas(canvas);
  }, []);

  const canvasRefCallback = useCallback((canvas: HTMLCanvasElement | null) => {
    if (canvasCanvas != null || canvas == null) return;
    setCanvasCanvas(canvas);
  }, []);

  return (
    <>
      <canvas ref={canvasRefCallback} {...props} />
      <canvas ref={glRefCallback} {...props} />
      <Aether.Composite path={path}>{bootstrapped && children}</Aether.Composite>
    </>
  );
};

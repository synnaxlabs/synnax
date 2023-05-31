import {
  CanvasHTMLAttributes,
  DetailedHTMLProps,
  ReactElement,
  useCallback,
  useEffect,
  useState,
} from "react";

import { Box } from "@synnaxlabs/x";

import { Bob } from "../bob/main";
import { useResize } from "../hooks";

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
  const bootstrap = Bob.useBootstrap<Bootstrap>();
  const {
    path,
    state: [, setCanvas],
  } = Bob.useComponent<WorkerCanvasProps>(
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
      <canvas ref={glRefCallback} {...props} />
      <canvas ref={canvasRefCallback} {...props} />
      <Bob.Composite path={path}>{bootstrapped && children}</Bob.Composite>
    </>
  );
};

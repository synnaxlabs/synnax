import { createContext, useContext } from "react";

export type PIDEngine = "canvas" | "jsx";

export interface PIDContext {
  render: <P>(type: string, props: P) => ReactElement | null;
}

export interface CanvasRenderContext {
  canvas: CanvasRenderingContext2D;
}

export type contextRenderer = <P>(type: string, props: P) => ReactElement | null;
export type canvasRenderer<P> = (ctx: CanvasRenderContext, props: P) => void;
export type jsxRenderer<P> = (props: P) => ReactElement;

export interface RenderSet<P> {
  canvas: canvasRenderer<P>;
  svg: jsxRenderer<P>;
}

export const PIDContextC = createContext<PIDContext>({
  render: () => null,
});

export const usePIDContext = (): PIDContext => {
  const ctx = useContext(PIDContextC);
  if (ctx == null)
    throw new Error("usePIDContext must be used within a PIDContextProvider");
  return ctx;
};

export const workerRenderer = (worker: Worker | null): contextRenderer => {
  return <P extends unknown>(type: string, props: P): ReactElement | null => {
    if (worker == null) return null;
    worker.postMessage({ type, props });
    return null;
  };
};

export const renderCanvas = (
  canvas: CanvasRenderingContext2D,
  registry: Record<string, RenderSet<unknown>>
): contextRenderer => {
  return <P extends unknown>(type: string, props: P): ReactElement | null => {
    const renderer = registry[type]?.canvas;
    if (renderer == null) return null;
    renderer({ canvas }, props);
    return null;
  };
};

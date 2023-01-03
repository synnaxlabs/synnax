import { createContext, PropsWithChildren, useContext, useRef } from "react";

import { RenderingEngine, RenderRequest } from "../render/engine";
import { RGBATuple, XY } from "../render/line";

export interface Line {
  x: Float32Array;
  y: Float32Array;
  scale: XY;
  offset: XY;
  color: RGBATuple;
}

export interface CanvasContextValue {
  render: RenderF;
}

type RenderF = (req: RenderRequest) => void;

const CanvasContext = createContext<CanvasContextValue | null>(null);

export const useRenderer = (): RenderF => {
  const ctx = useContext(CanvasContext);
  if (ctx == null) throw new Error("Canvas context not found");
  return ctx.render;
};

export interface CanvasProps extends PropsWithChildren<any> {}

export const Canvas = ({ children }: CanvasProps): JSX.Element => {
  const engineRef = useRef<RenderingEngine | null>(null);

  const render = (req: RenderRequest): void => {
    if (engineRef.current == null) return;
    engineRef.current.render(req);
  };

  return (
    <CanvasContext.Provider value={{ render }}>
      <canvas
        ref={(e) => {
          engineRef.current = new RenderingEngine(e as HTMLCanvasElement);
        }}
        style={{
          width: "100%",
          height: "100%",
          position: "absolute",
        }}
      />
      {children}
    </CanvasContext.Provider>
  );
};

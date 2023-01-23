import {
  CanvasHTMLAttributes,
  createContext,
  DetailedHTMLProps,
  useContext,
  useState,
} from "react";

import { newDefaultRendererRegistry } from "../registry";
import { GLContext } from "../renderer";

export interface GLCanvasContextValue {
  ctx: GLContext | null;
}

type HTMLCanvasProps = DetailedHTMLProps<
  CanvasHTMLAttributes<HTMLCanvasElement>,
  HTMLCanvasElement
>;

export const GLCanvasContext = createContext<GLCanvasContextValue | null>(null);

export interface GLCanvasProps extends Omit<HTMLCanvasProps, "ref"> {}

export const useGLContext = (): GLContext | null =>
  useContext(GLCanvasContext)?.ctx ?? null;

export const GLCanvas = ({ children, ...props }: GLCanvasProps): JSX.Element | null => {
  const [ctx, setCtx] = useState<GLContext | null>(null);

  const createCanvas = (e: HTMLCanvasElement | null): void => {
    if (e == null || ctx !== null) return;
    setCtx(new GLContext(e, newDefaultRendererRegistry()));
  };

  return (
    <GLCanvasContext.Provider value={{ ctx }}>
      <canvas ref={createCanvas} {...props} />
      {children}
    </GLCanvasContext.Provider>
  );
};

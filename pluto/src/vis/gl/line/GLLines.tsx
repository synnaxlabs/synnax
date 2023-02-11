import { useEffect } from "react";

import { useGLContext } from "../GLCanvas/GLCanvas";

import { GLLine, LineRenderRequest } from "./renderer";

import { Box, XY } from "@/spatial";

export interface GLLinesProps {
  box: Box;
  lines: GLLine[];
  clearOverscan?: XY;
}

export const GLLines = ({ box, lines, clearOverscan }: GLLinesProps): null => {
  const ctx = useGLContext();
  useEffect(() => {
    if (ctx == null) return;
    const base = ctx.registry.get<LineRenderRequest>("line");
    const scissor = ctx.scissor(base, clearOverscan);
    scissor.render(ctx, { box, lines });
    return () => scissor.clear(ctx, box);
  }, [box, lines]);
  return null;
};

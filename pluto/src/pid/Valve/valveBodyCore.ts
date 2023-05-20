import { dimensions, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { CanvasRenderContext } from "../PIDContext";

import { hex } from "@/color";
import { Canvas } from "@/vis/canvas";

export const valveBodySpec = {
  path: "M1 3.23317V46.7668C1 48.2529 2.56328 49.2199 3.89299 48.5564L50.107 25.4956C50.6693 25.215 51.3307 25.215 51.893 25.4956L98.107 48.5564C99.4367 49.2199 101 48.2529 101 46.7668V3.23317C101 1.74711 99.4367 0.780079 98.107 1.4436L51.893 24.5044C51.3307 24.785 50.6693 24.785 50.107 24.5044L3.893 1.4436C2.56329 0.78008 1 1.74711 1 3.23317Z",
  dimensions: { width: 102, height: 50 },
};

export const valveBodyProps = z.object({
  id: z.string(),
  position: xy,
  dimensions,
  fill: hex,
  stroke: hex,
});

export type ValveBodyProps = z.infer<typeof valveBodyProps>;

export const valveBodyCanvas = (
  { canvas }: CanvasRenderContext,
  { position, dimensions: { width, height }, fill, stroke }: ValveBodyProps
): void => {
  const path = new Path2D();
  let mat = new DOMMatrix();
  mat = Canvas.translate(mat, position);
  mat = Canvas.scale(mat, {
    width: width / valveBodySpec.dimensions.width,
    height: height / valveBodySpec.dimensions.height,
  });
  path.addPath(new Path2D(valveBodySpec.path), mat);
  canvas.strokeStyle = stroke;
  canvas.fillStyle = fill;
  canvas.lineWidth = 1;
  canvas.fill(path);
  canvas.stroke(path);
};

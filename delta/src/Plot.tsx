import { useEffect, useRef, useState } from "react";

import { LineRenderer } from "./features/visualization/render/line";

export const Plot = (): JSX.Element => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [gl, setGl] = useState<WebGLRenderingContext | null>(null);
  const [width, setWidth] = useState(1504);
  const [height, setHeight] = useState(1608);

  useEffect(() => {
    if (canvasRef.current != null) {
      const gl = canvasRef.current.getContext("webgl", {
        antialias: true,
        transparent: false,
      }) as WebGLRenderingContext;
      setWidth(canvasRef.current.width);
      setHeight(canvasRef.current.height);
      setGl(gl);
    }
  }, []);

  useEffect(() => {
    if (gl != null) {
      render(gl);
    }
  }, [gl]);

  return (
    <div>
      <canvas
        ref={canvasRef}
        width={width}
        height={height}
        style={{
          position: "fixed",
          width: "100%",
          height: "100%",
          zIndex: -1,
          top: 0,
          left: 0,
        }}
      />
      <h1 style={{ left: 1000, top: 500, position: "relative" }}>Hello</h1>
    </div>
  );
};

const pointCount = 1e7;
const now = performance.now();
const yData = Float32Array.from(
  { length: pointCount },
  (_, i) => -1 + Math.sin(i / (pointCount / 2))
);
const xData = Float32Array.from(
  { length: pointCount },
  (_, i) => -1 + i / (pointCount / 2)
);

const render = (gl: WebGLRenderingContext): void => {
  resizeCanvasToDisplaySize(gl.canvas);

  gl.clear(gl.COLOR_BUFFER_BIT);

  const r = new LineRenderer(gl);

  r.render({
    x: xData,
    y: yData,
    color: [1, 1, 1, 1],
    scale: {
      x: 1,
      y: 1,
    },
    rootOffset: {
      x: 0,
      y: 0,
    },
  });

  r.destroy();
};

function resizeCanvasToDisplaySize(canvas) {
  // Lookup the size the browser is displaying the canvas in CSS pixels.
  const displayWidth = canvas.clientWidth;
  const displayHeight = canvas.clientHeight;

  // Check if the canvas is not the same size.
  const needResize = canvas.width !== displayWidth || canvas.height !== displayHeight;

  if (needResize) {
    // Make the canvas the same size
    canvas.width = displayWidth;
    canvas.height = displayHeight;
  }

  return needResize;
}

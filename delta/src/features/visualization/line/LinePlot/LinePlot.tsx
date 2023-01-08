// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useRef } from "react";

import { CSSBox, useResize } from "@synnaxlabs/pluto";

import { useRenderingContext } from "../../components/Canvas";
import { LineRenderRequest } from "../../render/line";
import { RenderingContext } from "../../render/render";
import { LinePlotVS } from "../types";

import { useAsyncEffect } from "@/hooks";

import "./LinePlot.css";

export interface LinePlotProps {
  vis: LinePlotVS;
  onChange: (vis: LinePlotVS) => void;
  resizeDebounce: number;
}

export const LinePlot = ({
  vis,
  onChange,
  resizeDebounce,
}: LinePlotProps): JSX.Element => {
  const ref = useRef<HTMLDivElement | null>(null);

  const ctx = useRenderingContext();

  const render = useCallback(
    async (ctx: RenderingContext, vis: LinePlotVS, el: HTMLDivElement | null) => {
      if (el == null) return;
      if (
        vis.channels.y1.length === 0 ||
        vis.ranges.x1.length === 0 ||
        vis.channels.x1 === ""
      )
        return;

      const box = new CSSBox(el.getBoundingClientRect());
      const renderer = ctx.registry.get<LineRenderRequest>("line");
      await renderer.render(ctx, { box, range: vis.ranges.x1[0], lines });
    },
    []
  );

  useAsyncEffect(async () => {
    if (ctx == null || ref.current == null) return;
    await render(ctx, vis, ref.current);
  }, [vis, ctx]);

  useResize({
    ref,
    debounce: resizeDebounce,
    onResize: (): void => {
      if (ctx == null) return;
      void render(ctx, vis, ref.current).catch(console.error);
    },
  });

  return (
    <div className="delta-line-plot__container">
      <div ref={ref} className="delta-line-plot__plot" />
    </div>
  );
};

// const CorePlot = ({ width, height, left, top }: UseSizeReturn): JSX.Element => {
//   const render = useRenderingContext();

//   const [zoomMask, setZoomMask] = useState<PointBox | null>(null);

//   const onZoomDrag = (e: MouseEvent): void =>
//     setZoomMask((prev) => ({
//       ...(prev ?? { one: { x: e.clientX, y: e.clientY } }),
//       two: {
//         x: e.clientX,
//         y: e.clientY,
//       },
//     }));

//   const dragProps = useDrag({
//     onMove: onZoomDrag,
//     onEnd: () => {
//       setZoomMask(null);
//     },
//   });

//   const zoomMaskStyle: CSSProperties | null = {
//     position: "fixed",
//     backgroundColor: "rgba(0, 0, 0, 0.2)",
//   };

//   if (zoomMask != null) {
//     zoomMaskStyle.width = Math.abs(zoomMask.one.x - zoomMask.two.x);
//     zoomMaskStyle.height = Math.abs(zoomMask.one.y - zoomMask.two.y);
//     zoomMaskStyle.left = Math.min(zoomMask.one.x, zoomMask.two.x);
//     zoomMaskStyle.top = Math.min(zoomMask.one.y, zoomMask.two.y);
//     if (zoomMaskStyle.height < 35) {
//       zoomMaskStyle.height = height;
//       zoomMaskStyle.top = top;
//     } else if (zoomMaskStyle.width < 35) {
//       zoomMaskStyle.width = width;
//       zoomMaskStyle.left = left;
//     }
//   }

//   return (
//     <>
//       <div style={{ width, height }} onMouseDown={dragProps.onDragStart} />;
//       {zoomMaskStyle != null && <div style={zoomMaskStyle} />}
//     </>
//   );
// };

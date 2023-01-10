// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useRef } from "react";

import { TArray, SampleValue, TimeStamp } from "@synnaxlabs/client";
import { CSSBox, hexToRGBA, useResize, useMergedRef, Box } from "@synnaxlabs/pluto";
import * as d3 from "d3";

import { useCanvasContext } from "../../components/Canvas";
import { LineRenderRequest } from "../../gl/line";
import { RenderingContext } from "../../gl/render";
import { EnhancedLinePlotVS, LinePlotVS } from "../types";

import { useSelectTheme } from "@/features/layout";
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
  const ref = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const ctx = useCanvasContext();

  const theme = useSelectTheme();

  const render = useCallback(
    async (ctx: RenderingContext, vis: LinePlotVS, box: Box) => {
      const oldBox = box;
      box = box.resize({ x: 50, y: 50 }).translate({ x: 25, y: 10 });
      const r = ctx.registry.get<LineRenderRequest>("line");
      if (
        vis.channels.y1.length === 0 ||
        vis.ranges.x1.length === 0 ||
        vis.channels.x1 === ""
      ) {
        r.render(ctx, { box, lines: [] });
        return;
      }

      const vs = new EnhancedLinePlotVS(vis);
      const { ranges, keys } = vs;
      const data = await ctx.client.retrieve({ ranges, keys });
      const y1Data = data.filter(({ key }) => vis.channels.y1.includes(key));
      const y1Arrays = y1Data.flatMap(({ arrays }) => arrays);
      const y1Max = y1Arrays.reduce(
        (acc: SampleValue, arr: TArray) => (arr.max > acc ? arr.max : acc),
        -Infinity
      ) as number;
      const y1Min = y1Arrays.reduce(
        (acc: SampleValue, arr: TArray) => (arr.min < acc ? arr.min : acc),
        Infinity
      ) as number;
      const y1Scale = 1 / (y1Max - y1Min);
      const y1Offset = -y1Min * y1Scale;
      const xData = data.find(({ key }) => key === vis.channels.x1);
      if (xData == null) return;
      const xArrays = xData.arrays[0];
      const xScale = 1 / (xArrays.range as number);
      const xOffset = -(xArrays.min as number) * xScale;

      const lines = y1Data.map(({ key, glBuffers, arrays }, i) => ({
        color: [
          ...hexToRGBA(theme?.colors.visualization.palettes.default[i])
            .slice(0, 3)
            .map((c) => c / 255),
          1,
        ],
        scale: {
          x: xScale,
          y: y1Scale,
        },
        offset: {
          x: xOffset,
          y: y1Offset,
        },
        y: glBuffers[0],
        x: xData.glBuffers[0],
        strokeWidth: 2,
        length: arrays[0].length,
      }));

      r.render(ctx, {
        box,
        lines,
      });

      const svg = d3.select(svgRef.current);

      const d3XScale = d3
        .scaleTime()
        .domain([
          new TimeStamp(Number(xArrays.min) + xArrays.timeRange.start.valueOf()).date(),
          new TimeStamp(Number(xArrays.max) + xArrays.timeRange.start.valueOf()).date(),
        ])
        .range([25, box.width + 25])
        .nice();

      const xGrid = d3
        .axisBottom(d3XScale)
        .tickSizeInner(-box.height)
        .tickSizeOuter(0)
        .tickPadding(10)
        .ticks(25);

      const yScale = d3
        .scaleLinear()
        .domain([y1Min, y1Max])
        .range([box.height + 10, 10]);

      const yGrid = d3
        .axisLeft(yScale)
        .tickSizeInner(-box.width)
        .tickSizeOuter(0)
        .tickPadding(10)
        .ticks(25);

      // remove old y grid
      svg.select(".y-grid").remove();
      svg.select(".x-grid").remove();

      svg
        .append("g")
        .attr("class", "y-grid")
        .attr("transform", `translate(25, 0)`)
        .call(yGrid);

      // remove all text
      svg
        .append("g")
        .attr("class", "x-grid")
        .attr("transform", `translate(0, ${oldBox.height - 40})`)
        .call(xGrid);

      // change all colors to light gray
      svg.selectAll("line").attr("stroke", "var(--pluto-gray-m2)");
      svg.selectAll("path").attr("stroke", "var(--pluto-gray-m2");

      svg.selectAll("text").attr("fill", "var(--pluto-gray-p1)");
    },
    [theme]
  );

  const handleResize = useCallback(
    (box: Box): void => {
      if (ctx == null) return;
      void render(ctx, vis, box).catch(console.error);
    },
    [vis, ctx]
  );

  useAsyncEffect(async () => {
    if (ctx == null || ref.current == null) return;
    await render(ctx, vis, new CSSBox(ref.current.getBoundingClientRect()));
  }, [vis, ctx]);

  const resizeRef = useResize(handleResize, { debounce: 0 });

  const mergedRef = useMergedRef(ref, resizeRef);

  return (
    <div className="delta-line-plot__container">
      <div ref={mergedRef} className="delta-line-plot__plot">
        <svg ref={svgRef} className="delta-line-plot__svg" />
      </div>
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

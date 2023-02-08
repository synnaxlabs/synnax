// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useRef, useState } from "react";

import { TArray, SampleValue } from "@synnaxlabs/client";
import {
  hexToRGBA,
  useResize,
  useMergedRef,
  Box,
  AxisProps,
  GLLine,
  ZERO_BOX,
  GLLines,
  RGBATuple,
  Axis,
  ZoomPanMask,
  useZoomPan,
  Scale,
  Bound,
  DECIMAL_BOX,
} from "@synnaxlabs/pluto";

import { TelemetryClient, TelemetryClientResponse } from "../../../telem/client";
import { useTelemetryClient } from "../../../telem/TelemetryContext";

import { useSelectTheme } from "@/features/layout";

import { LineSVis } from "../types";

import { useAsyncEffect } from "@/hooks";

import "./LinePlot.css";

export interface LinePlotProps {
  vis: LineSVis;
  onChange: (vis: LineSVis) => void;
  resizeDebounce: number;
}

interface RenderingPackage {
  axes: AxisProps[];
  lines: GLLine[];
  glBox: Box;
}

export const LinePlot = ({
  vis,
  onChange,
  resizeDebounce,
}: LinePlotProps): JSX.Element => {
  const ref = useRef<HTMLDivElement>(null);
  const theme = useSelectTheme();
  const client = useTelemetryClient();
  const [pkg, setPkg] = useState<RenderingPackage>({
    axes: [],
    lines: [],
    glBox: ZERO_BOX,
  });
  const [zoom, setZoom] = useState<Box>(DECIMAL_BOX);

  const updateRenderingPackage = useCallback(
    async (vis: LineSVis, box: Box, zoom: Box): Promise<void> => {
      if (client == null) return;
      const data = await fetchData(vis, client);

      const y1Data = data.filter(({ key }) => vis.channels.y1.includes(key));
      const y1Scale = Scale.scale(calcBound(y1Data, 0.01))
        .scale(1)
        .translate(-zoom.bottom)
        .magnify(1 / zoom.height);

      const xData = data.filter(({ key }) => key === vis.channels.x1);
      if (xData.length === 0) return;

      const x1Scale = Scale.scale(calcBound(xData, 0))
        .scale(1)
        .translate(-zoom.left)
        .magnify(1 / zoom.width);

      const glBox = new Box(
        { x: box.left + 20, y: box.top + 20 },
        box.width - 40,
        box.height - 40
      );

      const lines = y1Data
        .map(({ key, glBuffers, arrays }, i) => {
          console.log(arrays.length);
          return arrays.map((arr, j) => ({
            color: [
              ...hexToRGBA(theme?.colors.visualization.palettes.default[i] as string)
                .slice(0, 3)
                .map((c) => c / 255),
              1,
            ] as RGBATuple,
            scale: {
              x: x1Scale.dim(1),
              y: y1Scale.dim(1),
            },
            offset: {
              x: x1Scale.pos(0),
              y: y1Scale.pos(0),
            },
            y: glBuffers[j],
            x: xData[0].glBuffers[j],
            strokeWidth: 3,
            length: arr.length,
          }));
        })
        .flat();

      setPkg({
        axes: [
          {
            location: "bottom",
            scale: x1Scale,
            type: "time",
            position: { y: box.height - 20, x: 20 },
            size: box.width - 40,
            height: box.height - 40,
            pixelsPerTick: 60,
            showGrid: true,
          },
          {
            location: "left",
            scale: y1Scale,
            position: { y: 20, x: 20 },
            height: box.width - 40,
            size: box.height - 40,
            showGrid: true,
          },
        ],
        lines,
        glBox,
      });
    },
    [theme, client, vis]
  );

  useAsyncEffect(async () => {
    if (ref.current == null) return;
    await updateRenderingPackage(vis, new Box(ref.current), zoom);
  }, [vis, client]);

  const handleResize = useCallback(
    (box: Box): void => {
      void updateRenderingPackage(vis, box, zoom);
    },
    [vis, zoom, client]
  );

  const zoomPanProps = useZoomPan({
    onChange: setZoom,
    panHotkey: "hift",
    allowPan: false,
    threshold: { width: 30, height: 30 },
    minZoom: { x: 0.02, y: 0.02 },
  });

  const resizeRef = useResize(handleResize, { debounce: 100 });

  const mergedRef = useMergedRef(ref, resizeRef);

  return (
    <div className="delta-line-plot__container">
      <div className="delta-line-plot__plot" ref={mergedRef}>
        <ZoomPanMask
          style={{
            position: "absolute",
            top: 20,
            left: 20,
            width: pkg.glBox.width,
            height: pkg.glBox.height,
          }}
          {...zoomPanProps}
        />
        <GLLines lines={pkg.lines} box={pkg.glBox} />
        <svg className="delta-line-plot__svg">
          {pkg.axes.map((axis, i) => (
            <Axis key={i} {...axis} />
          ))}
        </svg>
      </div>
    </div>
  );
};

const fetchData = async (
  vis: LineSVis,
  client: TelemetryClient
): Promise<TelemetryClientResponse[]> => {
  const keys = Object.values(vis.channels).flat();
  const ranges = Object.values(vis.ranges).flat();
  return await client.retrieve({ keys, ranges });
};

const calcBound = (data: TelemetryClientResponse[], padding: number): Bound => {
  const arrays = data.flatMap(({ arrays }) => arrays);
  const upper = Number(
    arrays.reduce(
      (acc: SampleValue, arr: TArray) => (arr.max > acc ? arr.max : acc),
      -Infinity
    )
  );
  const lower = Number(
    arrays.reduce(
      (acc: SampleValue, arr: TArray) => (arr.min < acc ? arr.min : acc),
      Infinity
    )
  );
  const _padding = (upper - lower) * padding;
  return { lower: lower - _padding, upper: upper + _padding };
};

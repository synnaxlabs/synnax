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
  ZERO_XY,
  ONE_XY,
} from "@synnaxlabs/pluto";

import { useTelemetryClient } from "../../../telem/TelemetryContext";
import { LineSVis } from "../types";

import { useSelectTheme } from "@/features/layout";
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
  const [zoom, setZoom] = useState<Box>(new Box(ZERO_XY, ONE_XY));

  const updateRenderingPackage = useCallback(
    async (vis: LineSVis, box: Box, zoom: Box): Promise<void> => {
      if (client == null) return;
      const data = await fetchData(vis, client);

      const y1Data = data.filter(({ key }) => vis.channels.y1.includes(key));
      const y1Range = calcRange(y1Data, 0.01);
      const y1Scale = 1 / (y1Range[1] - y1Range[0]) / zoom.height;
      const y1Offset = -y1Range[0] * y1Scale;

      const xData = data.filter(({ key }) => key === vis.channels.x1);
      if (xData.length === 0) return;
      const x1Range = calcRange(xData, 0);
      const x1Scale = 1 / (x1Range[1] - x1Range[0]) / zoom.width;
      const xOffset = -x1Range[0] * x1Scale;

      const glBox = box.translate({ x: 20, y: 20 }).resize({ x: 40, y: 40 });

      const lines = y1Data.map(({ key, glBuffers, arrays }, i) => ({
        color: [
          ...hexToRGBA(theme?.colors.visualization.palettes.default[i])
            .slice(0, 3)
            .map((c) => c / 255),
          1,
        ] as RGBATuple,
        scale: {
          x: x1Scale,
          y: y1Scale,
        },
        offset: {
          x: xOffset - zoom.x / zoom.width,
          y: y1Offset - (1 - zoom.y - zoom.height) / zoom.height,
        },
        y: glBuffers[0],
        x: xData[0].glBuffers[0],
        strokeWidth: 3,
        length: arrays[0].length,
      }));

      setPkg({
        axes: [
          {
            location: "bottom",
            range: [
              x1Range[0] + (x1Range[1] - x1Range[0]) * zoom.x,
              x1Range[1] - (x1Range[1] - x1Range[0]) * (1 - zoom.x - zoom.width),
            ],
            type: "time",
            position: { y: box.height - 20, x: 20 },
            size: box.width - 40,
            height: box.height - 40,
            pixelsPerTick: 60,
            showGrid: true,
          },
          {
            location: "left",
            range: [
              y1Range[0] + (y1Range[1] - y1Range[0]) * (1 - zoom.y - zoom.height),
              y1Range[1] - (y1Range[1] - y1Range[0]) * zoom.y,
            ],
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
    await updateRenderingPackage(
      vis,
      new Box(ref.current.getBoundingClientRect()),
      zoom
    );
  }, [vis, zoom, client]);

  const handleResize = useCallback(
    (box: Box): void => {
      void updateRenderingPackage(vis, box, zoom);
    },
    [vis, zoom, client]
  );

  const zoomPanProps = useZoomPan({
    onChange: setZoom,
    panHotkey: "Shift",
    threshold: { x: 30, y: 30 },
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

const calcRange = (
  data: TelemetryClientResponse[],
  padding: number
): [number, number] => {
  const arrays = data.flatMap(({ arrays }) => arrays);
  const max = Number(
    arrays.reduce(
      (acc: SampleValue, arr: TArray) => (arr.max > acc ? arr.max : acc),
      -Infinity
    )
  );
  const min = Number(
    arrays.reduce(
      (acc: SampleValue, arr: TArray) => (arr.min < acc ? arr.min : acc),
      Infinity
    )
  );
  const _padding = (max - min) * padding;
  return [min - _padding, max + _padding];
};

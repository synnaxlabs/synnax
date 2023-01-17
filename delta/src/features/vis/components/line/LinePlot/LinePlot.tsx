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
} from "@synnaxlabs/pluto";

import { LineSVis } from "../types";

import { useSelectTheme } from "@/features/layout";
import { useAsyncEffect } from "@/hooks";

import "./LinePlot.css";

import { useTelemetryClient } from "@/features/vis/telem/TelemetryContext";

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

  const updateRenderingPackage = useCallback(
    async (vis: LineSVis, box: Box): Promise<void> => {
      if (client == null) return;
      const data = await fetchData(vis, client);

      const y1Data = data.filter(({ key }) => vis.channels.y1.includes(key));
      const y1Range = calcRange(y1Data, 0.01);
      const y1Scale = 1 / (y1Range[1] - y1Range[0]);
      const y1Offset = -y1Range[0] * y1Scale;

      const xData = data.filter(({ key }) => key === vis.channels.x1);
      if (xData.length === 0) return;
      const x1Range = calcRange(xData, 0);
      const x1Scale = 1 / (x1Range[1] - x1Range[0]);
      const xOffset = -x1Range[0] * x1Scale;
      console.log(xOffset, y1Offset);

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
          x: xOffset + vis.pan.x / glBox.width,
          y: y1Offset - vis.pan.y / glBox.height,
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
            range: x1Range,
            type: "time",
            position: { y: box.height - 20, x: 20 },
            size: box.width - 40,
            height: box.height - 40,
            pixelsPerTick: 60,
            showGrid: true,
          },
          {
            location: "left",
            range: y1Range,
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
    await updateRenderingPackage(vis, new Box(ref.current.getBoundingClientRect()));
  }, [vis, client]);

  const handleResize = useCallback(
    (box: Box): void => {
      void updateRenderingPackage(vis, box);
    },
    [vis, client]
  );

  const resizeRef = useResize(handleResize, { debounce: 100 });

  const mergedRef = useMergedRef(ref, resizeRef);

  const zoomPanProps = useZoomPan({
    threshold: { x: 50, y: 50 },
    zoomHotkey: "Shift",
    panHotkey: "",
    onChange: (zoom: Box, pan: Box) => {
      console.log(pan);
      onChange({
        ...vis,
        ranges: {},
        pan,
      });
    },
  });

  return (
    <div className="delta-line-plot__container">
      <div className="delta-line-plot__plot" ref={mergedRef}>
        <GLLines lines={pkg.lines} box={pkg.glBox} />
        <ZoomPanMask
          style={{
            position: "absolute",
            width: "100%",
            height: "100%",
          }}
          {...zoomPanProps}
        />
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

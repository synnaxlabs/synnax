// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useState, useTransition } from "react";

import { SampleValue } from "@synnaxlabs/client";
import {
  hexToRGBA,
  useResize,
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
  Theme,
  Space,
  Typography,
} from "@synnaxlabs/pluto";
import { addSamples } from "@synnaxlabs/x";

import { useSelectTheme } from "@/features/layout";
import { AxisKey, X_AXIS_KEYS, YAxisKey, Y_AXIS_KEYS } from "@/features/vis/types";

import { TelemetryClient, TelemetryClientResponse } from "../../../telem/client";

import { useAsyncEffect } from "@/hooks";

import { useTelemetryClient } from "../../../telem/TelemetryContext";
import { LineSVis } from "../types";

import "./LinePlot.css";

export interface LinePlotProps {
  vis: LineSVis;
  onChange: (vis: LineSVis) => void;
  resizeDebounce: number;
}

interface RenderingState {
  axes: AxisProps[];
  lines: GLLine[];
  glBox: Box;
}

interface DataState {
  data: LineVisData;
  error: Error | null;
}

const initialDataState = (): DataState => ({
  data: { ...ZERO_DATA },
  error: null,
});

export const LinePlot = ({
  vis,
  onChange,
  resizeDebounce,
}: LinePlotProps): JSX.Element => {
  const theme = useSelectTheme();
  const client = useTelemetryClient();
  const [data, setData] = useState<DataState>(initialDataState());
  const [pkg, setPkg] = useState<RenderingState>({
    axes: [],
    lines: [],
    glBox: ZERO_BOX,
  });
  const [box, setBox] = useState<Box>(ZERO_BOX);

  const [zoom, setZoom] = useState<Box>(DECIMAL_BOX);
  const [, startDraw] = useTransition();

  const valid = isValid(vis);

  useAsyncEffect(async () => {
    if (client == null || !valid) return setData(initialDataState());
    try {
      const data = await fetchData(vis, client);
      setData({ data, error: null });
    } catch (error) {
      setData({ data: { ...ZERO_DATA }, error: error as Error });
    }
  }, [vis, client]);

  useEffect(() => {
    if (data == null) return setPkg({ axes: [], lines: [], glBox: ZERO_BOX });
    const lines = buildGLLines(data.data, zoom, theme);
    const [axes, glBox] = buildAxes(data.data, zoom, box);
    startDraw(() => setPkg({ lines, axes, glBox }));
  }, [zoom, theme, box, data]);

  const zoomPanProps = useZoomPan({
    onChange: setZoom,
    panHotkey: "Shift",
    allowPan: true,
    threshold: { width: 30, height: 30 },
  });

  const handleResize = useCallback((box: Box) => setBox(box), [setBox]);

  const resizeRef = useResize(handleResize, { debounce: 100 });

  if (data.error != null)
    return (
      <Space.Centered>
        <Typography.Text
          level="h4"
          color="var(--pluto-error-z)"
          wrap
          style={{ padding: "2rem" }}
        >
          {data.error.message}
        </Typography.Text>
      </Space.Centered>
    );
  if (valid && data == null)
    return (
      <Space.Centered>
        <Typography.Text level="h4" color="var(--pluto-gray-m0)">
          Loading...
        </Typography.Text>
      </Space.Centered>
    );
  if (!valid)
    return (
      <Space.Centered>
        <Typography.Text level="h4" color="var(--pluto-gray-m0)">
          Invalid Visualization
        </Typography.Text>
      </Space.Centered>
    );
  if (valid && Object.values(data).flat().length === 0)
    return (
      <Space.Centered>
        <Typography.Text level="h4" color="var(--pluto-gray-m0)">
          No Data Found
        </Typography.Text>
      </Space.Centered>
    );

  return (
    <div className="delta-line-plot__container">
      <div className="delta-line-plot__plot" ref={resizeRef}>
        <ZoomPanMask
          style={{
            position: "absolute",
            top: 10,
            left: 30,
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

const ZERO_DATA: LineVisData = {
  y1: [],
  y2: [],
  y3: [],
  y4: [],
  x1: [],
  x2: [],
};

const fetchData = async (
  vis: LineSVis,
  client: TelemetryClient
): Promise<LineVisData> => {
  const keys = Object.values(vis.channels)
    .flat()
    .filter((key) => key.length > 0);
  const ranges = Object.values(vis.ranges).flat();
  const entries = await client.retrieve({ keys, ranges });
  const data: LineVisData = { ...ZERO_DATA };
  Object.values(vis.ranges).forEach((ranges) => {
    ranges.forEach((range) => {
      Object.entries(vis.channels).forEach(([axis, channelKeys]) => {
        if (!Array.isArray(channelKeys)) channelKeys = [channelKeys as string];
        data[axis as AxisKey] = data[axis as AxisKey].concat(
          entries.filter(
            ({ key, range: r }) => channelKeys.includes(key) && r === range
          )
        );
      });
    });
  });
  return data;
};

type LineVisData = Record<AxisKey, TelemetryClientResponse[]>;

const calcBound = (
  data: TelemetryClientResponse[],
  padding: number,
  includeOffset: boolean
): Bound => {
  const arrays = data.flatMap(({ arrays }) => arrays);
  const upper = Number(
    arrays.reduce((acc: SampleValue, arr) => {
      let max = arr.max;
      if (!includeOffset) max = addSamples(max, -arr.offset);
      return max > acc ? max : acc;
    }, -Infinity)
  );
  const lower = Number(
    arrays.reduce((acc: SampleValue, arr) => {
      let min = arr.min;
      if (!includeOffset) min = addSamples(min, -arr.offset);
      return min < acc ? min : acc;
    }, Infinity)
  );
  const _padding = (upper - lower) * padding;
  if (upper === lower) return { lower: lower - 1, upper: upper + 1 };
  return { lower: lower - _padding, upper: upper + _padding };
};

const buildGLLines = (data: LineVisData, zoom: Box, theme: Theme): GLLine[] => {
  const lines: GLLine[] = [];
  X_AXIS_KEYS.forEach((key) => {
    const xData = data[key];
    if (xData.length === 0) return;
    const xBound = calcBound(xData, 0, false);
    const xScale = Scale.scale(xBound)
      .scale(1)
      .translate(-zoom.left)
      .magnify(1 / zoom.width);
    Y_AXIS_KEYS.forEach((key) => {
      const yData = data[key];
      const yBound = calcBound(yData, 0.01, false);
      const yScale = Scale.scale(yBound)
        .scale(1)
        .translate(-zoom.bottom)
        .magnify(1 / zoom.height);
      const scale = { x: xScale.dim(1), y: yScale.dim(1) };
      const offset = { x: xScale.pos(0), y: yScale.pos(0) };
      yData.forEach((yRes, i) => {
        const xRes = xData[0];
        yRes.arrays.forEach((yArr, j) => {
          lines.push({
            color: [
              ...hexToRGBA(theme.colors.visualization.palettes.default[i])
                .slice(0, 3)
                .map((c) => c / 255),
              1,
            ] as RGBATuple,
            scale,
            offset,
            y: yRes.buffers.value[j].buf,
            x: xRes.buffers.value[j].buf,
            strokeWidth: 3,
            length: yArr.length,
          });
        });
      });
    });
  });
  return lines;
};

const AXIS_WIDTH = 20;
const BASE_AXIS_PADDING = 10;

const buildAxes = (data: LineVisData, zoom: Box, box: Box): [AxisProps[], Box] => {
  const axes: AxisProps[] = [];
  const leftYAxisWidth =
    ["y1", "y3"].filter((key) => data[key as YAxisKey].length > 0).length * AXIS_WIDTH +
    BASE_AXIS_PADDING;
  const rightYAxisWidth =
    ["y2", "y4"].filter((key) => data[key as YAxisKey].length > 0).length * AXIS_WIDTH +
    BASE_AXIS_PADDING;
  const topXAxisHeight = (data.x2.length > 0 ? 1 : 0) * AXIS_WIDTH + BASE_AXIS_PADDING;
  const bottomXAxisHeight =
    (data.x1.length > 0 ? 1 : 0) * AXIS_WIDTH + BASE_AXIS_PADDING;
  X_AXIS_KEYS.forEach((key, i) => {
    const res = data[key];
    if (res.length === 0) return;
    const location = key === "x1" ? "bottom" : "top";
    const bound = calcBound(res, 0, true);
    const scale = Scale.scale(bound)
      .scale(1)
      .translate(-zoom.left)
      .magnify(1 / zoom.width);
    const y =
      location === "top"
        ? BASE_AXIS_PADDING
        : box.height - BASE_AXIS_PADDING - AXIS_WIDTH;
    axes.push({
      location,
      position: { y, x: leftYAxisWidth },
      size: box.width - leftYAxisWidth - rightYAxisWidth,
      pixelsPerTick: 40,
      showGrid: i === 0,
      scale,
      height: box.height - topXAxisHeight - bottomXAxisHeight,
      type: "time",
    });
  });

  Y_AXIS_KEYS.forEach((key, i) => {
    const res = data[key];
    if (res.length === 0) return;
    const location = ["y1", "y3"].includes(key) ? "left" : "right";
    const bound = calcBound(res, 0.01, true);
    const scale = Scale.scale(bound)
      .scale(1)
      .translate(-zoom.bottom)
      .magnify(1 / zoom.height);
    axes.push({
      location,
      position: {
        x:
          location === "left"
            ? BASE_AXIS_PADDING + AXIS_WIDTH
            : box.width - AXIS_WIDTH - BASE_AXIS_PADDING,
        y: topXAxisHeight,
      },
      size: box.height - topXAxisHeight - bottomXAxisHeight,
      pixelsPerTick: 40,
      showGrid: key === "y1",
      height: box.width - leftYAxisWidth - rightYAxisWidth,
      scale,
    });
  });

  return [
    axes,
    new Box(
      { x: box.left + leftYAxisWidth, y: box.top + topXAxisHeight },
      box.width - leftYAxisWidth - rightYAxisWidth,
      box.height - topXAxisHeight - bottomXAxisHeight
    ),
  ];
};

const isValid = (vis: LineSVis): boolean => {
  const hasRanges = X_AXIS_KEYS.some((key) => {
    const v = vis.ranges[key];
    return v?.length > 0;
  });
  const hasXAxis = X_AXIS_KEYS.some((key) => {
    const v = vis.channels[key];
    return v != null && v.length > 0;
  });
  const hasYAxis = Y_AXIS_KEYS.some((key) => {
    const v = vis.channels[key];
    return v?.length > 0;
  });
  return hasRanges && hasXAxis && hasYAxis;
};

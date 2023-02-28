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
import { Icon } from "@synnaxlabs/media";
import {
  hexToRGBA,
  useResize,
  Box,
  AxisProps,
  GLLine,
  ZERO_BOX,
  GLLines,
  Axis,
  Viewport,
  UseViewportHandler,
  Scale,
  Bound,
  DECIMAL_BOX,
  Theme,
  Space,
  Typography,
  Menu as PMenu,
  ZERO_BOUND,
  Divider,
} from "@synnaxlabs/pluto";
import { addSamples, TimeRange } from "@synnaxlabs/x";

import { AxisKey, X_AXIS_KEYS, YAxisKey, Y_AXIS_KEYS } from "../../../../vis/types";
import { TelemetryClient, TelemetryClientResponse } from "../../../telem/client";
import { useTelemetryClient } from "../../../telem/TelemetryContext";
import { LineSVis } from "../types";

import { Menu } from "@/components";
import { useSelectTheme } from "@/features/layout";
import { useAsyncEffect } from "@/hooks";

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
  xBound: Bound;
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
    xBound: ZERO_BOUND,
  });
  const [box, setBox] = useState<Box>(ZERO_BOX);

  const [zoom, setZoom] = useState<Box>(DECIMAL_BOX);
  const [selection, setSelection] = useState<Box | null>(null);
  const [, startDraw] = useTransition();
  const [tick, setTick] = useState(0);

  const valid = isValid(vis);

  useEffect(() => {
    const i = setInterval(() => {
      setTick((t) => t + 1);
    }, 2000);
    return () => clearInterval(i);
  }, []);

  useAsyncEffect(async () => {
    if (client == null || !valid) return setData(initialDataState());
    try {
      const data = await fetchData(vis, client);
      setData({ data, error: null });
    } catch (error) {
      setData({ data: { ...ZERO_DATA }, error: error as Error });
    }
  }, [vis, tick, client]);

  useEffect(() => {
    if (theme == null) return;
    if (data == null)
      return setPkg({ axes: [], lines: [], glBox: ZERO_BOX, xBound: ZERO_BOUND });
    const lines = buildGLLines(data.data, zoom, theme);
    const [xBound, axes, glBox] = buildAxes(data.data, zoom, box);
    startDraw(() => setPkg({ lines, axes, glBox, xBound }));
  }, [zoom, theme, box, data]);

  const menuProps = PMenu.useContextMenu();

  const handleZoomPanSelect: UseViewportHandler = useCallback(
    ({ box, mode, cursor }) => {
      if (mode === "select") {
        setSelection(box);
        return menuProps.open(cursor);
      }
      setSelection(null);
      setZoom(box);
    },
    []
  );

  const viewportProps = Viewport.use({
    onChange: handleZoomPanSelect,
  });

  const handleResize = useCallback((box: Box) => setBox(box), [setBox]);

  const resizeRef = useResize(handleResize, { debounce: 100 });

  if (data.error != null)
    return (
      <Space.Centered>
        <Typography.Text
          level="h4"
          color="var(--pluto-error-z)"
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

  const ContextMenu = (): JSX.Element => {
    const getTimeRange = (): TimeRange => {
      console.log("TR");
      if (selection == null) throw new Error("Selection is null");
      const scale = Scale.scale(pkg.xBound)
        .scale(1)
        .translate(-selection.left)
        .magnify(1 / selection.width)
        .reverse();
      return new TimeRange(scale.pos(0), scale.pos(1));
    };

    return (
      <PMenu>
        {selection !== null && (
          <>
            <PMenu.Item
              size="small"
              itemKey="copyPython"
              startIcon={<Icon.Python />}
              onClick={() => {
                const tr = getTimeRange();
                const code = `synnax.TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`;
                void navigator.clipboard.writeText(code);
              }}
            >
              Copy Time Range as Python
            </PMenu.Item>
            <PMenu.Item
              size="small"
              itemKey="copyTS"
              onClick={() => {
                const tr = getTimeRange();
                const code = `new TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`;
                void navigator.clipboard.writeText(code);
              }}
            >
              Copy Time Range as TypeScript
            </PMenu.Item>
            <PMenu.Item
              size="small"
              itemKey="copyTS"
              startIcon={<Icon.Time />}
              onClick={() => {
                const tr = getTimeRange();
                const code = `${tr.start.fString("ISO", "local")} ${tr.end.fString(
                  "ISO",
                  "local"
                )}`;
                void navigator.clipboard.writeText(code);
              }}
            >
              Copy Time Range as ISO
            </PMenu.Item>
            <Divider direction="x" padded />
          </>
        )}
        <Menu.Item.HardReload />
      </PMenu>
    );
  };

  return (
    <PMenu.ContextMenu
      className="delta-line-plot__container"
      {...menuProps}
      menu={() => <ContextMenu />}
    >
      <div className="delta-line-plot__plot" ref={resizeRef}>
        <Viewport.Mask
          style={{
            position: "absolute",
            top: 10,
            left: 30,
            width: pkg.glBox.width,
            height: pkg.glBox.height,
          }}
          {...viewportProps}
        />
        <GLLines lines={pkg.lines} box={pkg.glBox} />
        <svg className="delta-line-plot__svg">
          {pkg.axes.map((axis, i) => (
            <Axis key={i} {...axis} />
          ))}
        </svg>
      </div>
    </PMenu.ContextMenu>
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
  const entries = await client.retrieve({ keys, ranges, bypassCache: true });
  const data: LineVisData = { ...ZERO_DATA };
  Object.values(vis.ranges).forEach((ranges) =>
    ranges.forEach((range) =>
      Object.entries(vis.channels).forEach(([axis, channelKeys]) => {
        if (!Array.isArray(channelKeys)) channelKeys = [channelKeys as string];
        data[axis as AxisKey] = data[axis as AxisKey].concat(
          entries.filter(
            ({ key, range: r }) => channelKeys.includes(key) && r === range
          )
        );
      })
    )
  );
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
    const xBound = calcBound(xData, 0, true);
    const xScale = Scale.scale(xBound)
      .scale(1)
      .translate(-zoom.left)
      .magnify(1 / zoom.width);
    Y_AXIS_KEYS.forEach((key) => {
      const yData = data[key];
      const yBound = calcBound(yData, 0.01, true);
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
            color: hexToRGBA(theme.colors.visualization.palettes.default[i], 1, 255),
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

const AXIS_WIDTH = 15;
const BASE_AXIS_PADDING = 12.5;

const buildAxes = (
  data: LineVisData,
  zoom: Box,
  box: Box
): [Bound, AxisProps[], Box] => {
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

  let xBound = ZERO_BOUND;
  X_AXIS_KEYS.forEach((key, i) => {
    const res = data[key];
    if (res.length === 0) return;
    const location = key === "x1" ? "bottom" : "top";
    xBound = calcBound(res, 0, false);
    const xScale = Scale.scale(xBound)
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
      scale: xScale,
      height: box.height - topXAxisHeight - bottomXAxisHeight,
      type: "time",
    });
  });

  Y_AXIS_KEYS.forEach((key) => {
    const res = data[key];
    if (res.length === 0) return;
    const location = ["y1", "y3"].includes(key) ? "left" : "right";
    const bound = calcBound(res, 0.01, false);
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
    xBound,
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

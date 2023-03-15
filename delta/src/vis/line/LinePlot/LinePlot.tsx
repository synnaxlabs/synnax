// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useState } from "react";

import {
  GLLines,
  Axis,
  Viewport as PViewport,
  Menu as PMenu,
  useResize,
  Status,
  Rule,
  RuleAnnotationProps,
  Theming,
} from "@synnaxlabs/pluto";
import { XY, Box, ZERO_BOX } from "@synnaxlabs/x";

import { Viewport } from "../viewport";

import { useSelectTheme } from "@/layout";
import { XAxisKey, X_AXIS_KEYS } from "@/vis/axis";
import { Axes } from "@/vis/line/axes";
import { Bounds } from "@/vis/line/bounds";
import { Channels } from "@/vis/line/channels";
import { Data } from "@/vis/line/data";
import { ContextMenu } from "@/vis/line/LinePlot/ContextMenu";
import { Lines } from "@/vis/line/lines";
import { Ranges } from "@/vis/line/ranges";
import { Scales } from "@/vis/line/scales";

import "./LinePlot.css";

interface HoverState {
  cursor: XY;
  box: Box;
}

export const LinePlot = ({ layoutKey }: { layoutKey: string }): JSX.Element => {
  const theme = useSelectTheme();

  const [container, setContainer] = useState<Box>(ZERO_BOX);

  const { viewportProps, menuProps, viewport, selection, hover } =
    Viewport.use(layoutKey);

  const channels = Channels.use(layoutKey);
  const ranges = Ranges.use(layoutKey);
  const data = Data.use(channels, ranges);
  const bounds = Bounds.use(data, 0.01);
  const scales = Scales.use(bounds, viewport);
  const axes = Axes.use(container, scales);
  const lines = Lines.use(container, data, scales, axes, theme);
  // console.log("LinePlot", { channels, ranges, data, bounds, scales, axes, lines });

  const handleResize = useCallback((box: Box) => setContainer(box), [setContainer]);

  const resizeRef = useResize(handleResize, { debounce: 100 });

  if (data.error != null)
    return (
      <Status.Text.Centered level="h4" variant="error" hideIcon>
        {data.error.message}
      </Status.Text.Centered>
    );
  const valid = [channels, ranges].every((x) => x.valid);
  if (data.loading)
    return (
      <Status.Text.Centered level="h4" variant="loading" hideIcon>
        Loading...
      </Status.Text.Centered>
    );
  if (!valid)
    return (
      <Status.Text.Centered level="h4" variant="disabled" hideIcon>
        Invalid Visualization
      </Status.Text.Centered>
    );
  if (data.empty)
    return (
      <Status.Text.Centered level="h4" variant="disabled" hideIcon>
        No Data Found
      </Status.Text.Centered>
    );

  return (
    <PMenu.ContextMenu
      className="delta-line-plot__container"
      {...menuProps}
      menu={() => <ContextMenu scale={scales.decimal("x1")} selection={selection} />}
    >
      <div className="delta-line-plot__plot pluto--no-select" ref={resizeRef}>
        <PViewport.Mask
          style={{ position: "absolute", ...axes.innerBox.css }}
          {...viewportProps}
        />
        <GLLines lines={lines.gl} box={lines.box} />
        <svg className="delta-line-plot__svg">
          {Object.entries(axes.axes).map(([key, axis]) => (
            <Axis key={key} {...axis} />
          ))}
        </svg>
        <svg className="delta-line-plot__svg pluto--no-select" style={{ zIndex: 2 }}>
          <Tooltip
            hover={hover}
            scales={scales}
            data={data}
            container={container}
            axes={axes}
            channels={channels}
          />
        </svg>
      </div>
    </PMenu.ContextMenu>
  );
};

interface TooltipProps {
  container: Box;
  hover: HoverState | null;
  scales: Scales;
  data: Data;
  axes: Axes;
  channels: Channels;
}

export const Tooltip = ({
  hover,
  scales,
  axes,
  data,
  container,
  channels,
}: TooltipProps): JSX.Element => {
  if (hover == null) return <></>;
  const annotation: RuleAnnotationProps[] = [];
  let arrayIndex: number | null = null;
  let position: number = 0;
  let value: number = 0;

  const { theme } = Theming.useContext();

  const xScale = scales.decimal("x1")?.reverse();
  if (xScale == null) return <></>;
  const scalePos = xScale.pos(hover.box.left);

  Object.values(data.axis("x1")).forEach((res) => {
    res.arrays.forEach((arr, j) => {
      if (arrayIndex != null) return;
      const pos = arr.binarySearch(BigInt(scalePos));
      if (pos !== -1) {
        arrayIndex = j;
        value = Number(arr.data[pos]);
        position = pos;
      }
    });
  });

  if (arrayIndex == null) return <></>;

  const left = scales.normal("x1")?.pos(value) as number;

  data.forEachChannel((key, axis, data) => {
    if (X_AXIS_KEYS.includes(axis as XAxisKey)) return;
    const ch = channels.get(key);
    const scale = scales.normal(axis);
    if (scale == null || ch == null) return;
    Object.values(data).forEach((res, i) => {
      const value = res.arrays[arrayIndex as number]?.data[position];
      if (value == null) return;
      annotation.push({
        values: {
          [ch.name]: value.toString(),
        },
        stroke: theme.colors.visualization.palettes.default[i],
        position: (1 - scale.pos(value as number)) * container.height,
      });
    });
  });

  try {
    return (
      <Rule
        direction="y"
        position={left * container.width}
        size={{
          upper: axes.innerBox.height + axes.innerBox.top,
          lower: axes.innerBox.top,
        }}
        annotations={annotation}
      />
    );
  } catch (e) {
    console.error(e);
    return <></>;
  }
};

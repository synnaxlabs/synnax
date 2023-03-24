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
  StatusTextProps,
} from "@synnaxlabs/pluto";
import { XY, Box, ZERO_BOX } from "@synnaxlabs/x";
import { Theme } from "@tauri-apps/api/window";

import { StatusProvider } from "../core";
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
  const bounds = Bounds.use(layoutKey, data, 0.01);
  const scales = Scales.use(bounds, viewport);
  const axes = Axes.use(container, scales);
  const lines = Lines.use(container, data, scales, axes, theme);

  const handleResize = useCallback((box: Box) => setContainer(box), [setContainer]);

  const resizeRef = useResize(handleResize, { debounce: 100 });

  for (const item of [channels, ranges, data] as StatusProvider[]) {
    if (item.status.display) {
      const s = item.status as StatusTextProps;
      return (
        <Status.Text.Centered level="h4" variant={s.variant} hideIcon>
          {s.children}
        </Status.Text.Centered>
      );
    }
  }

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
            axes={axes}
            channels={channels}
          />
        </svg>
      </div>
    </PMenu.ContextMenu>
  );
};

interface TooltipProps {
  hover: HoverState | null;
  scales: Scales;
  data: Data;
  axes: Axes;
  channels: Channels;
  theme: Theme;
}

export const Tooltip = ({
  hover,
  scales,
  axes,
  data,
  channels,
}: TooltipProps): JSX.Element => {
  if (hover == null) return <></>;
  const annotation: RuleAnnotationProps[] = [];
  let arrayIndex: number | null = null;
  let position: number = 0;
  let value: number = 0;

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

  data.forEachChannel((key, axis, responses) => {
    if (X_AXIS_KEYS.includes(axis as XAxisKey)) return;
    const ch = channels.get(key);
    const scale = scales.normal(axis);
    if (scale == null || ch == null) return;
    Object.values(responses).forEach((res) => {
      const value = res.arrays[arrayIndex as number]?.data[position];
      if (value == null) return;
      annotation.push({
        key: ch.key,
        values: {
          [ch.name]: value.toString(),
        },
        position:
          (1 - scale.pos(value as number)) * axes.innerBox.height + axes.innerBox.top,
      });
    });
  });

  try {
    return (
      <Rule
        direction="y"
        position={left * axes.innerBox.width + axes.innerBox.left}
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

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Bounds, Location, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { TypographyLevel } from "@/core";
import { Color } from "@/core/color";
import {
  LinePlot as CoreLinePlot,
  LinePlotProps as CoreLinePlotProps,
} from "@/core/vis/LinePlot";
import { RangeTelem } from "@/telem/range/main";

export const axisProps = z.object({
  id: z.string(),
  location: Location.strictOuterZ,
  label: z.string().optional(),
  bounds: Bounds.looseZ.optional(),
  color: Color.z.optional(),
  showGrid: z.boolean().optional(),
});

export type AxisProps = z.input<typeof axisProps>;
export type ParsedAxisProps = z.output<typeof axisProps>;

export const coreLineProps = z.object({
  axes: z.object({
    x: z.string(),
    y: z.string(),
  }),
  channels: z.object({
    x: z.number().optional(),
    y: z.number(),
  }),
  color: Color.z,
  strokeWidth: z.number().optional(),
  label: z.string().optional(),
  downsample: z.number().optional(),
});

export const staticLineProps = coreLineProps.extend({
  variant: z.literal("static"),
  range: TimeRange.z,
});

export type StaticLineProps = z.input<typeof staticLineProps>;
export type ParsedStaticLineProps = z.output<typeof staticLineProps>;

export const dynamicLineProps = coreLineProps.extend({
  variant: z.literal("dynamic"),
  span: TimeSpan.z,
});

export type DynamicLineProps = z.input<typeof dynamicLineProps>;
export type ParsedDynamicLineProps = z.output<typeof dynamicLineProps>;

const lineProps = z.union([staticLineProps, dynamicLineProps]);

export type LineProps = z.input<typeof lineProps>;
type ParsedLineProps = z.output<typeof lineProps>;

export const linePlotProps = z.object({
  axes: z.array(axisProps),
  lines: z.array(lineProps),
});

export interface LinePlotProps extends CoreLinePlotProps {
  axes: AxisProps[];
  lines: LineProps[];
  title?: string;
  showTitle?: boolean;
  onTitleChange?: (value: string) => void;
  titleLevel?: TypographyLevel;
  showLegend?: boolean;
}

export const LinePlot = ({
  lines: pLines,
  axes: pAxes,
  showTitle = true,
  title = "",
  onTitleChange,
  showLegend = true,
  titleLevel = "h4",
  ...restProps
}: LinePlotProps): ReactElement => {
  const { axes, lines } = linePlotProps.parse({ axes: pAxes, lines: pLines });
  const xAxes = axes.filter(({ location }) => location.isY);
  return (
    <CoreLinePlot {...restProps}>
      {xAxes.map((a, i) => (
        <XAxis
          key={a.id}
          {...a}
          index={i}
          lines={lines.filter((l) => l.axes.x === a.id)}
          yAxes={axes.filter(({ location }) => location.isX)}
        />
      ))}
      {showLegend && <CoreLinePlot.Legend />}
      {showTitle && (
        <CoreLinePlot.Title value={title} onChange={onTitleChange} level={titleLevel} />
      )}
    </CoreLinePlot>
  );
};

interface XAxisProps extends ParsedAxisProps {
  lines: ParsedLineProps[];
  yAxes: ParsedAxisProps[];
  index: number;
}

const XAxis = ({
  yAxes,
  lines,
  showGrid,
  index,
  ...props
}: XAxisProps): ReactElement => {
  return (
    <CoreLinePlot.XAxis type="time" {...props} showGrid={showGrid ?? index === 0}>
      {yAxes.map((a, i) => (
        <YAxis
          key={a.id}
          {...a}
          lines={lines.filter((l) => l.axes.y === a.id)}
          showGrid={showGrid ?? (index === 0 && i === 0)}
        ></YAxis>
      ))}
    </CoreLinePlot.XAxis>
  );
};

interface YAxisProps extends ParsedAxisProps {
  lines: ParsedLineProps[];
}

const lineKey = ({ channels: { x, y } }: LineProps): string => `${x ?? 0}-${y}`;

const YAxis = ({ lines, ...props }: YAxisProps): ReactElement => {
  return (
    <CoreLinePlot.YAxis type="linear" {...props}>
      {lines.map((l) =>
        l.variant === "static" ? (
          <StaticLine key={lineKey(l)} {...l} />
        ) : (
          <DynamicLine key={lineKey(l)} {...l} />
        )
      )}
    </CoreLinePlot.YAxis>
  );
};

const DynamicLine = ({
  span,
  channels: { x, y },
  ...props
}: ParsedDynamicLineProps): ReactElement => {
  const telem = RangeTelem.useDynamicXY({ span, x, y });
  return <CoreLinePlot.Line telem={telem} {...props} />;
};

const StaticLine = ({
  range,
  channels: { x, y },
  ...props
}: ParsedStaticLineProps): ReactElement => {
  const telem = RangeTelem.useXY({ timeRange: range, x, y });
  return <CoreLinePlot.Line telem={telem} {...props} />;
};

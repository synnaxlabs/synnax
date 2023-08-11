// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Bounds, Box, Direction, Location, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { TypographyLevel, UseViewportProps } from "@/core";
import { Color } from "@/core/color";
import {
  LinePlot as CoreLinePlot,
  LinePlotProps as CoreLinePlotProps,
} from "@/core/vis/LinePlot";
import { Measure } from "@/core/vis/Measure/Measure";
import { Tooltip } from "@/core/vis/Tooltip/Tooltip";
import { RemoteTelem } from "@/telem/remote/main";

export const axisProps = z.object({
  id: z.string(),
  location: Location.strictOuterZ,
  label: z.string().optional(),
  labelDirection: Direction.looseZ.optional(),
  bounds: Bounds.looseZ.optional(),
  color: Color.z.optional(),
  showGrid: z.boolean().optional(),
  type: z.union([z.literal("time"), z.literal("linear")]),
});

export type AxisProps = z.input<typeof axisProps>;
export type ParsedAxisProps = z.output<typeof axisProps>;

export const coreLineProps = z.object({
  id: z.string(),
  axes: z.object({
    x: z.string(),
    y: z.string(),
  }),
  channels: z.object({
    x: z.number().optional(),
    y: z.number(),
  }),
  color: Color.z,
  strokeWidth: z.number().optional().default(1),
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

export const ruleProps = z.object({
  id: z.string(),
  position: z.number(),
  color: Color.z,
  axis: z.string(),
  label: z.string(),
  lineWidth: z.number().min(1).optional(),
  lineDash: z.number().optional(),
  units: z.string().optional(),
});

export type RuleProps = z.input<typeof ruleProps>;
type ParsedRuleProps = z.output<typeof ruleProps>;

export const linePlotProps = z.object({
  axes: z.array(axisProps),
  lines: z.array(lineProps),
  rules: z.array(ruleProps).optional(),
});

export interface LinePlotProps extends CoreLinePlotProps {
  axes: AxisProps[];
  lines: LineProps[];
  rules?: RuleProps[];
  title?: string;
  showTitle?: boolean;
  onTitleChange?: (value: string) => void;
  titleLevel?: TypographyLevel;
  showLegend?: boolean;
  onLineLabelChange?: (id: string, value: string) => void;
  onLineColorChange?: (id: string, value: Color) => void;
  onRuleLabelChange?: (id: string, value: string) => void;
  onRulePositionChange?: (id: string, value: number) => void;
  enableTooltip?: boolean;
  enableMeasure?: boolean;
  initialViewport?: UseViewportProps["initial"];
  onViewportChange?: UseViewportProps["onChange"];
  viewportTriggers?: UseViewportProps["triggers"];
}

export const LinePlot = ({
  lines: pLines,
  axes: pAxes,
  showTitle = true,
  title = "",
  onTitleChange,
  showLegend = true,
  titleLevel = "h4",
  onLineLabelChange,
  onLineColorChange,
  onRuleLabelChange,
  onRulePositionChange,
  rules: pRules,
  enableTooltip = true,
  enableMeasure = false,
  initialViewport = Box.DECIMAL,
  onViewportChange,
  viewportTriggers,
  ...restProps
}: LinePlotProps): ReactElement => {
  const { axes, lines, rules } = linePlotProps.parse({
    axes: pAxes,
    lines: pLines,
    rules: pRules,
  });
  const xAxes = axes.filter(({ location }) => location.isY);
  return (
    <CoreLinePlot {...restProps}>
      {xAxes.map((a, i) => {
        const _lines = lines.filter((l) => l.axes.x === a.id);
        const _axes = axes.filter(({ location }) => location.isX);
        const _rules = rules?.filter((r) =>
          [..._axes.map(({ id }) => id), a.id].includes(r.axis)
        );
        return (
          <XAxis
            key={a.id}
            {...a}
            index={i}
            lines={_lines}
            yAxes={_axes}
            rules={_rules}
            onRuleLabelChange={onRuleLabelChange}
            onRulePositionChange={onRulePositionChange}
          />
        );
      })}
      {showLegend && (
        <CoreLinePlot.Legend
          onLabelChange={onLineLabelChange}
          onColorChange={onLineColorChange}
        />
      )}
      {showTitle && (
        <CoreLinePlot.Title value={title} onChange={onTitleChange} level={titleLevel} />
      )}
      <CoreLinePlot.Viewport
        initial={initialViewport}
        onChange={onViewportChange}
        triggers={viewportTriggers}
      >
        {enableTooltip && <Tooltip />}
        {enableMeasure && <Measure />}
      </CoreLinePlot.Viewport>
    </CoreLinePlot>
  );
};

interface XAxisProps extends ParsedAxisProps {
  lines: ParsedLineProps[];
  yAxes: ParsedAxisProps[];
  rules?: ParsedRuleProps[];
  onRuleLabelChange?: (id: string, value: string) => void;
  onRulePositionChange?: (id: string, value: number) => void;
  index: number;
}

const XAxis = ({
  yAxes,
  lines,
  showGrid,
  index,
  rules,
  onRuleLabelChange,
  onRulePositionChange,
  ...props
}: XAxisProps): ReactElement => {
  const _rules = rules?.filter((r) => r.axis === props.id);
  return (
    <CoreLinePlot.XAxis {...props} showGrid={showGrid ?? index === 0}>
      {yAxes.map((a, i) => {
        const lines_ = lines.filter((l) => l.axes.y === a.id);
        const rules_ = rules?.filter((r) => r.axis === a.id);
        return (
          <YAxis
            key={a.id}
            {...a}
            lines={lines_}
            rules={rules_}
            showGrid={showGrid ?? (index === 0 && i === 0)}
            onRuleLabelChange={onRuleLabelChange}
            onRulePositionChange={onRulePositionChange}
          />
        );
      })}
      {_rules?.map((r) => (
        <CoreLinePlot.Rule
          aetherKey={r.id}
          key={r.id}
          {...r}
          onLabelChange={(value) => onRuleLabelChange?.(r.id, value)}
          onPositionChange={(value) => onRulePositionChange?.(r.id, value)}
        />
      ))}
    </CoreLinePlot.XAxis>
  );
};

interface YAxisProps extends ParsedAxisProps {
  lines: ParsedLineProps[];
  rules?: ParsedRuleProps[];
  onRuleLabelChange?: (id: string, value: string) => void;
  onRulePositionChange?: (id: string, value: number) => void;
}

const lineKey = ({ channels: { x, y } }: LineProps): string => `${x ?? 0}-${y}`;

const YAxis = ({
  lines,
  rules,
  onRuleLabelChange,
  onRulePositionChange,
  ...props
}: YAxisProps): ReactElement => {
  return (
    <CoreLinePlot.YAxis {...props}>
      {lines.map((l) =>
        l.variant === "static" ? (
          <StaticLine key={lineKey(l)} {...l} />
        ) : (
          <DynamicLine key={lineKey(l)} {...l} />
        )
      )}
      {rules?.map((r) => (
        <CoreLinePlot.Rule
          aetherKey={r.id}
          key={r.id}
          {...r}
          onLabelChange={(value) => onRuleLabelChange?.(r.id, value)}
          onPositionChange={(value) => onRulePositionChange?.(r.id, value)}
        />
      ))}
    </CoreLinePlot.YAxis>
  );
};

const DynamicLine = ({
  span,
  channels: { x, y },
  ...props
}: ParsedDynamicLineProps): ReactElement => {
  const telem = RemoteTelem.useDynamicXY({ span, x, y });
  return <CoreLinePlot.Line telem={telem} {...props} />;
};

const StaticLine = ({
  range,
  id,
  channels: { x, y },
  ...props
}: ParsedStaticLineProps): ReactElement => {
  const telem = RemoteTelem.useXY({ timeRange: range, x, y });
  return <CoreLinePlot.Line aetherKey={id} telem={telem} {...props} />;
};

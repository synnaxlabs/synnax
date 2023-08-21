// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import {
  Box,
  CrudeBounds,
  CrudeDirection,
  CrudeLocation,
  Location,
  TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";

import { Color } from "@/color";
import { Remote } from "@/telem/remote";
import { Text } from "@/text";
import { Viewport } from "@/viewport";
import { axis } from "@/vis/axis";
import { LinePlot as Core } from "@/vis/lineplot";
import { Measure } from "@/vis/measure/Measure";
import { Rule } from "@/vis/rule";
import { Tooltip } from "@/vis/tooltip/Tooltip";

export interface AxisProps {
  keyX: string;
  location: CrudeLocation;
  label: string;
  labelDirection?: CrudeDirection;
  bounds?: CrudeBounds;
  color: Color.Crude;
  showGrid?: boolean;
  type: axis.TickType;
}

export interface BaseLineProps {
  keyX: string;
  axes: {
    x: string;
    y: string;
  };
  channels: {
    y: number;
    x?: number;
  };
  color: Color.Crude;
  strokeWidth?: number;
  label?: string;
  downsample?: number;
}

export interface StaticLineProps extends BaseLineProps {
  variant: "static";
  range: TimeRange;
}

export interface DynamicLineProps extends BaseLineProps {
  variant: "dynamic";
  span: TimeSpan;
}

export type LineProps = StaticLineProps | DynamicLineProps;

export interface RuleProps {
  id: string;
  position: number;
  color: Color.Crude;
  axis: string;
  label: string;
  lineWidth?: number;
  lineDash?: number;
  units?: string;
}
export interface LinePlotProps extends Core.LinePlotProps {
  axes: AxisProps[];
  lines: LineProps[];
  rules?: RuleProps[];
  title?: string;
  showTitle?: boolean;
  onTitleChange?: (value: string) => void;
  titleLevel?: Text.Level;
  showLegend?: boolean;
  onLineLabelChange?: (id: string, value: string) => void;
  onLineColorChange?: (id: string, value: Color.Color) => void;
  onRuleLabelChange?: (id: string, value: string) => void;
  onRulePositionChange?: (id: string, value: number) => void;
  enableTooltip?: boolean;
  enableMeasure?: boolean;
  initialViewport?: Viewport.UseProps["initial"];
  onViewportChange?: Viewport.UseProps["onChange"];
  viewportTriggers?: Viewport.UseProps["triggers"];
}

export const LinePlot = ({
  lines,
  axes,
  showTitle = true,
  title = "",
  onTitleChange,
  showLegend = true,
  titleLevel = "h4",
  onLineLabelChange,
  onLineColorChange,
  onRuleLabelChange,
  onRulePositionChange,
  rules,
  enableTooltip = true,
  enableMeasure = false,
  initialViewport = Box.DECIMAL,
  onViewportChange,
  viewportTriggers,
  ...restProps
}: LinePlotProps): ReactElement => {
  const xAxes = axes.filter(({ location }) => new Location(location).isY);
  return (
    <Core.LinePlot {...restProps}>
      {xAxes.map((a, i) => {
        const _lines = lines.filter((l) => l.axes.x === a.keyX);
        const _axes = axes.filter(({ location }) => new Location(location).isX);
        const _rules = rules?.filter((r) =>
          [..._axes.map(({ keyX }) => keyX), a.keyX].includes(r.axis)
        );
        return (
          <XAxis
            key={a.keyX}
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
        <Core.Legend
          onLabelChange={onLineLabelChange}
          onColorChange={onLineColorChange}
        />
      )}
      {showTitle && (
        <Core.Title value={title} onChange={onTitleChange} level={titleLevel} />
      )}
      <Core.Viewport
        initial={initialViewport}
        onChange={onViewportChange}
        triggers={viewportTriggers}
      >
        {enableTooltip && <Tooltip />}
        {enableMeasure && <Measure />}
      </Core.Viewport>
    </Core.LinePlot>
  );
};

interface XAxisProps extends AxisProps {
  lines: LineProps[];
  yAxes: AxisProps[];
  rules?: RuleProps[];
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
  const _rules = rules?.filter((r) => r.axis === props.keyX);
  return (
    <Core.XAxis {...props} showGrid={showGrid ?? index === 0}>
      {yAxes.map((a, i) => {
        const lines_ = lines.filter((l) => l.axes.y === a.keyX);
        const rules_ = rules?.filter((r) => r.axis === a.keyX);
        return (
          <YAxis
            key={a.keyX}
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
        <Rule.Rule
          aetherKey={r.id}
          key={r.id}
          {...r}
          onLabelChange={(value) => onRuleLabelChange?.(r.id, value)}
          onPositionChange={(value) => onRulePositionChange?.(r.id, value)}
        />
      ))}
    </Core.XAxis>
  );
};

interface YAxisProps extends AxisProps {
  lines: LineProps[];
  rules?: RuleProps[];
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
    <Core.YAxis {...props}>
      {lines.map((l) =>
        l.variant === "static" ? (
          <StaticLine key={lineKey(l)} {...l} />
        ) : (
          <DynamicLine key={lineKey(l)} {...l} />
        )
      )}
      {rules?.map((r) => (
        <Rule.Rule
          aetherKey={r.id}
          key={r.id}
          {...r}
          onLabelChange={(value) => onRuleLabelChange?.(r.id, value)}
          onPositionChange={(value) => onRulePositionChange?.(r.id, value)}
        />
      ))}
    </Core.YAxis>
  );
};

const DynamicLine = ({
  span,
  channels: { x, y },
  ...props
}: DynamicLineProps): ReactElement => {
  const telem = Remote.useDynamicXYSource({ span, x, y });
  return <Core.Line telem={telem} {...props} />;
};

const StaticLine = ({
  range,
  keyX,
  channels: { x, y },
  ...props
}: StaticLineProps): ReactElement => {
  const telem = Remote.useXYSource({ timeRange: range, x, y });
  return <Core.Line aetherKey={keyX} telem={telem} {...props} />;
};

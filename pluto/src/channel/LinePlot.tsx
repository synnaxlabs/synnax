// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { type channel } from "@synnaxlabs/client";
import {
  box,
  type bounds,
  type direction,
  location,
  type TimeRange,
  type TimeSpan,
} from "@synnaxlabs/x";

import { HAUL_TYPE } from "@/channel/types";
import { type Color } from "@/color";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { Remote } from "@/telem/remote";
import { type Text } from "@/text";
import { type Viewport } from "@/viewport";
import { type axis } from "@/vis/axis";
import { LinePlot as Core } from "@/vis/lineplot";
import { Measure } from "@/vis/measure";
import { Rule } from "@/vis/rule";
import { Tooltip } from "@/vis/tooltip";

import "@/channel/LinePlot.css";

export interface AxisProps {
  id: string;
  location: location.Crude;
  label: string;
  labelDirection?: direction.Crude;
  bounds?: bounds.Crude;
  color: Color.Crude;
  showGrid?: boolean;
  type: axis.TickType;
}

export interface BaseLineProps {
  id: string;
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
  onAxisChannelDrop?: (axis: string, channels: channel.Key[]) => void;
  enableTooltip?: boolean;
  enableMeasure?: boolean;
  initialViewport?: Viewport.UseProps["initial"];
  onViewportChange?: Viewport.UseProps["onChange"];
  viewportTriggers?: Viewport.UseProps["triggers"];
}

const canDrop = Haul.canDropOfType(HAUL_TYPE);

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
  onAxisChannelDrop,
  rules,
  enableTooltip = true,
  enableMeasure = false,
  initialViewport = box.DECIMAL,
  onViewportChange,
  viewportTriggers,
  ...restProps
}: LinePlotProps): ReactElement => {
  const xAxes = axes.filter(({ location: l }) => location.isY(l));
  return (
    <Core.LinePlot {...restProps}>
      {xAxes.map((a, i) => {
        const _lines = lines.filter((l) => l.axes.x === a.id);
        const _axes = axes.filter(({ location: l }) => location.isX(l));
        const _rules = rules?.filter((r) =>
          [..._axes.map(({ id }) => id), a.id].includes(r.axis),
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
            onAxisChannelDrop={onAxisChannelDrop}
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
        {enableTooltip && <Tooltip.Tooltip />}
        {enableMeasure && <Measure.Measure />}
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
  onAxisChannelDrop?: (axis: string, channels: channel.Key[]) => void;
  index: number;
}

const XAxis = ({
  id,
  yAxes,
  lines,
  showGrid,
  index,
  rules,
  onRuleLabelChange,
  onRulePositionChange,
  onAxisChannelDrop,
  ...props
}: XAxisProps): ReactElement => {
  const dropProps = Haul.useDrop({
    type: "Channel.LinePlot.XAxis",
    canDrop,
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        onAxisChannelDrop?.(
          id,
          dropped.map(({ key }) => key as channel.Key),
        );
        return dropped;
      },
      [id, onAxisChannelDrop],
    ),
  });

  const _rules = rules?.filter((r) => r.axis === id);
  return (
    <Core.XAxis
      {...props}
      {...dropProps}
      showGrid={showGrid ?? index === 0}
      className={CSS(
        CSS.dropRegion(Haul.canDropOfType(HAUL_TYPE)(Haul.useDraggingState())),
      )}
    >
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
            onAxisChannelDrop={onAxisChannelDrop}
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
  onAxisChannelDrop?: (axis: string, channels: channel.Key[]) => void;
}

const lineKey = ({ channels: { x, y } }: LineProps): string => `${x ?? 0}-${y}`;

const YAxis = ({
  id,
  lines,
  rules,
  onRuleLabelChange,
  onRulePositionChange,
  onAxisChannelDrop,
  ...props
}: YAxisProps): ReactElement => {
  const dropProps = Haul.useDrop({
    type: "Channel.LinePlot.YAxis",
    canDrop,
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        onAxisChannelDrop?.(
          id,
          dropped.map(({ key }) => key as channel.Key),
        );
        return dropped;
      },
      [id, onAxisChannelDrop],
    ),
  });

  const dragging = Haul.useDraggingState();

  return (
    <Core.YAxis
      {...props}
      {...dropProps}
      className={CSS(CSS.dropRegion(Haul.canDropOfType(HAUL_TYPE)(dragging)))}
    >
      {lines.map((l) => (
        <Line key={lineKey(l)} {...l} />
      ))}
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

const Line = (props: LineProps): ReactElement =>
  props.variant === "static" ? <StaticLine {...props} /> : <DynamicLine {...props} />;

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
  id,
  channels: { x, y },
  ...props
}: StaticLineProps): ReactElement => {
  const telem = Remote.useXYSource({ timeRange: range, x, y });
  return <Core.Line aetherKey={id} telem={telem} {...props} />;
};

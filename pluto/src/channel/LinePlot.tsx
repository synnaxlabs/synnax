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
  location as loc,
  type TimeRange,
  type TimeSpan,
  type location,
} from "@synnaxlabs/x";

import { HAUL_TYPE } from "@/channel/types";
import { type Color } from "@/color";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { Remote } from "@/telem/remote";
import { type Text } from "@/text";
import { type Viewport } from "@/viewport";
import { LinePlot as Core } from "@/vis/lineplot";
import { Tooltip } from "@/vis/lineplot/tooltip";
import { Measure } from "@/vis/measure";
import { Rule } from "@/vis/rule";

import "@/channel/LinePlot.css";

/** Props for an axis in {@link LinePlot} */
export interface AxisProps extends Core.AxisProps {
  /** A unique identifier for the axis */
  key: string;
}

export interface BaseLineProps {
  key: string;
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
  key: string;
  position: number;
  color: Color.Crude;
  axis: string;
  label: string;
  lineWidth?: number;
  lineDash?: number;
  units?: string;
}

export interface LinePlotProps extends Core.LinePlotProps {
  // Axes
  axes: AxisProps[];
  onAxisChannelDrop?: (axis: string, channels: channel.Key[]) => void;
  onAxisChange?: (axis: Partial<AxisProps> & { key: string }) => void;
  // Lines
  lines: LineProps[];
  onLineChange?: (line: Partial<LineProps> & { key: string }) => void;
  // Rules
  rules?: RuleProps[];
  onRuleChange?: (rule: Partial<RuleProps> & { key: string }) => void;
  // Title
  title?: string;
  showTitle?: boolean;
  onTitleChange?: (value: string) => void;
  titleLevel?: Text.Level;
  // Legend
  showLegend?: boolean;
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
  onLineChange,
  onRuleChange,
  onAxisChannelDrop,
  onAxisChange,
  rules,
  enableTooltip = true,
  enableMeasure = false,
  initialViewport = box.DECIMAL,
  onViewportChange,
  viewportTriggers,
  ...restProps
}: LinePlotProps): ReactElement => {
  const xAxes = axes.filter(({ location: l }) => loc.isY(l));
  return (
    <Core.LinePlot {...restProps}>
      {xAxes.map((a, i) => {
        const axisLines = lines.filter((l) => l.axes.x === a.key);
        const yAxes = axes.filter(({ location: l }) => loc.isX(l));
        const axisRules = rules?.filter((r) =>
          [...yAxes.map(({ key: id }) => id), a.key].includes(r.axis),
        );
        return (
          <XAxis
            key={a.key}
            axis={a}
            index={i}
            lines={axisLines}
            yAxes={yAxes}
            rules={axisRules}
            onAxisChannelDrop={onAxisChannelDrop}
            onAxisChange={onAxisChange}
          />
        );
      })}
      {showLegend && (
        <Core.Legend
          onLabelChange={(key, label) => onLineChange?.({ key, label })}
          onColorChange={(key, color) => onLineChange?.({ key, color })}
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

interface XAxisProps
  extends Pick<
    LinePlotProps,
    "onRuleChange" | "lines" | "rules" | "onAxisChannelDrop" | "onAxisChange"
  > {
  axis: AxisProps;
  yAxes: AxisProps[];
  index: number;
}

const XAxis = ({
  yAxes,
  lines,
  index,
  rules,
  onRuleChange,
  onAxisChannelDrop,
  onAxisChange,
  axis: { location, key, showGrid, ...axis },
}: XAxisProps): ReactElement => {
  const dropProps = Haul.useDrop({
    type: "Channel.LinePlot.XAxis",
    canDrop,
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        onAxisChannelDrop?.(
          key,
          dropped.map(({ key }) => key as channel.Key),
        );
        return dropped;
      },
      [key, onAxisChannelDrop],
    ),
  });

  const xRules = rules?.filter((r) => r.axis === key);
  return (
    <Core.XAxis
      {...axis}
      {...dropProps}
      location={location as location.Y}
      showGrid={showGrid ?? index === 0}
      className={CSS(
        CSS.dropRegion(Haul.canDropOfType(HAUL_TYPE)(Haul.useDraggingState())),
      )}
      onAutoBoundsChange={(bounds) => onAxisChange?.({ key, bounds })}
    >
      {yAxes.map((a, i) => {
        const yLines = lines.filter((l) => l.axes.y === a.key);
        const yRules = rules?.filter((r) => r.axis === a.key);
        return (
          <YAxis
            key={a.key}
            axis={{
              ...a,
              showGrid: showGrid ?? (index === 0 && i === 0),
            }}
            lines={yLines}
            rules={yRules}
            onRuleChange={onRuleChange}
            onAxisChannelDrop={onAxisChannelDrop}
            onAxisChange={onAxisChange}
          />
        );
      })}
      {xRules?.map((rule) => (
        <Rule.Rule
          aetherKey={rule.key}
          {...rule}
          key={rule.key}
          onLabelChange={(value) => onRuleChange?.({ key: rule.key, label: value })}
          onPositionChange={(value) =>
            onRuleChange?.({ key: rule.key, position: value })
          }
        />
      ))}
    </Core.XAxis>
  );
};

interface YAxisProps
  extends Pick<
    LinePlotProps,
    "onRuleChange" | "lines" | "rules" | "onAxisChannelDrop" | "onAxisChange"
  > {
  axis: AxisProps;
}

const lineKey = ({ channels: { x, y } }: LineProps): string => `${x ?? 0}-${y}`;

const YAxis = ({
  lines,
  rules,
  onRuleChange,
  onAxisChannelDrop,
  onAxisChange,
  axis: { key, location: loc, ...props },
}: YAxisProps): ReactElement => {
  const dropProps = Haul.useDrop({
    type: "Channel.LinePlot.YAxis",
    canDrop,
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        onAxisChannelDrop?.(
          key,
          dropped.map(({ key }) => key as channel.Key),
        );
        return dropped;
      },
      [key, onAxisChannelDrop],
    ),
  });

  const dragging = Haul.useDraggingState();

  return (
    <Core.YAxis
      {...props}
      {...dropProps}
      location={loc as loc.X}
      className={CSS(CSS.dropRegion(Haul.canDropOfType(HAUL_TYPE)(dragging)))}
      onAutoBoundsChange={(bounds) => onAxisChange?.({ key, bounds })}
    >
      {lines.map((l) => (
        <Line key={lineKey(l)} line={l} />
      ))}
      {rules?.map((r) => (
        <Rule.Rule
          aetherKey={r.key}
          {...r}
          key={r.key}
          onLabelChange={(value) => onRuleChange?.({ key: r.key, label: value })}
          onPositionChange={(value) => onRuleChange?.({ key: r.key, position: value })}
        />
      ))}
    </Core.YAxis>
  );
};

const Line = ({ line }: { line: LineProps }): ReactElement =>
  line.variant === "static" ? <StaticLine line={line} /> : <DynamicLine line={line} />;

const DynamicLine = ({
  line: {
    key,
    span,
    channels: { x, y },
    ...props
  },
}: {
  line: DynamicLineProps;
}): ReactElement => {
  const telem = Remote.useDynamicXYSource({ span, x, y });
  return <Core.Line aetherKey={key} telem={telem} {...props} />;
};

const StaticLine = ({
  line: {
    range,
    key,
    channels: { x, y },
    ...props
  },
}: {
  line: StaticLineProps;
}): ReactElement => {
  const telem = Remote.useXYSource({ timeRange: range, x, y });
  return <Core.Line aetherKey={key} telem={telem} {...props} />;
};

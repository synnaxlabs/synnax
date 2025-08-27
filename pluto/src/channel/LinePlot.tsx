// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/channel/LinePlot.css";

import { type channel } from "@synnaxlabs/client";
import { type color } from "@synnaxlabs/x";
import { box, location as loc, type xy } from "@synnaxlabs/x/spatial";
import { type TimeRange, type TimeSpan } from "@synnaxlabs/x/telem";
import { type ReactElement, useCallback, useEffect, useMemo, useRef } from "react";

import { HAUL_TYPE } from "@/channel/types";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { usePrevious } from "@/hooks";
import { telem } from "@/telem/aether";
import { type Text } from "@/text";
import { type Viewport } from "@/viewport";
import { LinePlot as Core } from "@/vis/lineplot";
import { Range } from "@/vis/lineplot/range";
import { Tooltip } from "@/vis/lineplot/tooltip";
import { Measure } from "@/vis/measure";
import { Rule } from "@/vis/rule";

/** Props for an axis in {@link LinePlot} */
export interface AxisProps extends Omit<Core.AxisProps, "axisKey"> {
  /** A unique identifier for the axis */
  key: string;
}

export interface BaseLineProps {
  key: string;
  axes: { x: string; y: string };
  channels: { y: channel.KeyOrName; x?: channel.KeyOrName };
  color: color.Crude;
  strokeWidth?: number;
  label?: string;
  downsample?: number;
  downsampleMode?: telem.DownsampleMode;
}

export interface StaticLineProps extends BaseLineProps {
  variant: "static";
  timeRange: TimeRange;
}

export interface DynamicLineProps extends BaseLineProps {
  variant: "dynamic";
  timeSpan: TimeSpan;
}

export type LineProps = StaticLineProps | DynamicLineProps;

export interface RuleProps {
  key: string;
  position: number;
  color: color.Crude;
  axis: string;
  label: string;
  lineWidth?: number;
  lineDash?: number;
  units?: string;
}

export interface LinePlotProps extends Core.LinePlotProps {
  axes: AxisProps[];
  onAxisChannelDrop?: (axis: string, channels: channel.Key[]) => void;
  onAxisChange?: (axis: Partial<AxisProps> & { key: string }) => void;
  lines: LineProps[];
  onLineChange?: (line: Partial<LineProps> & { key: string }) => void;
  rules?: RuleProps[];
  onRuleChange?: (rule: Partial<RuleProps> & { key: string }) => void;
  onSelectRule?: (key: string) => void;
  title?: string;
  showTitle?: boolean;
  onTitleChange?: (value: string) => void;
  titleLevel?: Text.Level;
  showLegend?: boolean;
  legendVariant?: Core.LegendProps["variant"];
  legendPosition?: xy.XY;
  onLegendPositionChange?: (value: xy.XY) => void;
  enableTooltip?: boolean;
  enableMeasure?: boolean;
  initialViewport?: Viewport.UseProps["initial"];
  onViewportChange?: Viewport.UseProps["onChange"];
  viewportTriggers?: Viewport.UseProps["triggers"];
  rangeAnnotationProvider?: Range.ProviderProps;
}

const canDrop = Haul.canDropOfType(HAUL_TYPE);

/**
 * A line plot component that automatically pulls data from specified channels and
 * displays it. Can be used to render both real-time and historical data.
 */
export const LinePlot = ({
  lines,
  axes,
  showTitle = true,
  title = "",
  onTitleChange,
  showLegend = true,
  legendPosition,
  onLegendPositionChange,
  titleLevel = "h4",
  onLineChange,
  onRuleChange,
  onAxisChannelDrop,
  onAxisChange,
  rules,
  enableTooltip = true,
  enableMeasure = false,
  initialViewport = box.DECIMAL,
  legendVariant,
  onViewportChange,
  viewportTriggers,
  rangeAnnotationProvider: annotationProvider,
  onSelectRule,
  children,
  ...rest
}: LinePlotProps): ReactElement => {
  const xAxes = axes.filter(({ location: l }) => loc.isY(l));
  const ref = useRef<Viewport.UseRefValue | null>(null);
  const prevLinesLength = usePrevious(lines.length);
  const prevHold = usePrevious(rest.hold);
  const shouldResetViewport =
    (prevLinesLength === 0 && lines.length !== 0) ||
    (prevHold === true && rest.hold === false);
  useEffect(() => {
    if (shouldResetViewport) ref.current?.reset();
  }, [shouldResetViewport]);
  return (
    <Core.LinePlot {...rest}>
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
            annotationProvider={annotationProvider}
            onRuleChange={onRuleChange}
            onSelectRule={onSelectRule}
          />
        );
      })}
      {showLegend && (
        <Core.Legend
          onLineChange={onLineChange}
          position={legendPosition}
          onPositionChange={onLegendPositionChange}
          variant={legendVariant}
        />
      )}
      {showTitle && (
        <Core.Title value={title} onChange={onTitleChange} level={titleLevel} />
      )}
      <Core.Viewport
        initial={initialViewport}
        onChange={onViewportChange}
        triggers={viewportTriggers}
        ref={ref}
      >
        {enableTooltip && <Tooltip.Tooltip />}
        {enableMeasure && <Measure.Measure />}
        {children}
      </Core.Viewport>
    </Core.LinePlot>
  );
};

interface XAxisProps
  extends Pick<
    LinePlotProps,
    | "onRuleChange"
    | "lines"
    | "rules"
    | "onAxisChannelDrop"
    | "onAxisChange"
    | "onSelectRule"
  > {
  axis: AxisProps;
  yAxes: AxisProps[];
  index: number;
  annotationProvider?: Range.ProviderProps;
}

const XAxis = ({
  yAxes,
  lines,
  index,
  rules,
  onRuleChange,
  onSelectRule,
  onAxisChannelDrop,
  onAxisChange,
  axis: { location, key, showGrid, ...axis },
  annotationProvider,
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
  const dragging = Haul.useDraggingState();
  return (
    <Core.XAxis
      {...axis}
      {...dropProps}
      location={location as loc.Y}
      axisKey={key}
      showGrid={showGrid ?? index === 0}
      className={CSS(CSS.dropRegion(Haul.canDropOfType(HAUL_TYPE)(dragging)))}
      onAutoBoundsChange={(bounds) => onAxisChange?.({ key, bounds })}
      onLabelChange={(value) => onAxisChange?.({ key, label: value })}
    >
      {yAxes.map((a, i) => {
        const yLines = lines.filter((l) => l.axes.y === a.key);
        const yRules = rules?.filter((r) => r.axis === a.key);
        return (
          <YAxis
            key={a.key}
            axis={{ ...a, showGrid: showGrid ?? (index === 0 && i === 0) }}
            lines={yLines}
            rules={yRules}
            onRuleChange={onRuleChange}
            onAxisChannelDrop={onAxisChannelDrop}
            onAxisChange={onAxisChange}
            onSelectRule={onSelectRule}
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
          onUnitsChange={(value) => onRuleChange?.({ key: rule.key, units: value })}
          onSelect={() => onSelectRule?.(rule.key)}
        />
      ))}
      <Range.Provider {...annotationProvider} />
    </Core.XAxis>
  );
};

interface YAxisProps
  extends Pick<
    LinePlotProps,
    | "onRuleChange"
    | "lines"
    | "rules"
    | "onAxisChannelDrop"
    | "onAxisChange"
    | "onSelectRule"
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
  onSelectRule,
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
      axisKey={key}
      className={CSS(CSS.dropRegion(Haul.canDropOfType(HAUL_TYPE)(dragging)))}
      onAutoBoundsChange={(bounds) => onAxisChange?.({ key, bounds })}
      onLabelChange={(value) => onAxisChange?.({ key, label: value })}
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
          onUnitsChange={(value) => onRuleChange?.({ key: r.key, units: value })}
          onClick={() => onSelectRule?.(r.key)}
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
    timeSpan,
    channels: { x, y },
    axes,
    ...rest
  },
}: {
  line: DynamicLineProps;
}): ReactElement => {
  const { xTelem, yTelem } = useMemo(() => {
    const keepFor = Number(timeSpan.valueOf()) * 3;
    const yTelem = telem.streamChannelData({
      timeSpan,
      channel: y,
      keepFor,
    });
    const hasX = x != null && x !== 0;
    const xTelem = telem.streamChannelData({
      timeSpan,
      channel: hasX ? x : y,
      useIndexOfChannel: !hasX,
      keepFor,
    });
    return { xTelem, yTelem };
  }, [timeSpan.valueOf(), x, y]);
  return (
    <Core.Line
      key={key}
      aetherKey={key}
      y={yTelem}
      x={xTelem}
      legendGroup={axes.y.toUpperCase()}
      {...rest}
    />
  );
};

const StaticLine = ({
  line: {
    timeRange,
    key,
    channels: { x, y },
    ...rest
  },
}: {
  line: StaticLineProps;
}): ReactElement => {
  const { xTelem, yTelem } = useMemo(() => {
    const yTelem = telem.channelData({ timeRange, channel: y });
    const hasX = x != null && x !== 0;
    const xTelem = telem.channelData({
      timeRange,
      channel: hasX ? x : y,
      useIndexOfChannel: !hasX,
    });
    return { xTelem, yTelem };
  }, [timeRange.start.valueOf(), timeRange.end.valueOf(), x, y]);
  return (
    <Core.Line
      key={key}
      aetherKey={key}
      y={yTelem}
      x={xTelem}
      legendGroup={rest.axes.y.toUpperCase()}
      {...rest}
    />
  );
};

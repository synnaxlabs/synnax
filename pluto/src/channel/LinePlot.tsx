// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x/box";
import { color } from "@synnaxlabs/x/color";
import { direction } from "@synnaxlabs/x/direction";
import { location as loc } from "@synnaxlabs/x/location";
import { telem as xtelem } from "@synnaxlabs/x/telem";
import { xy } from "@synnaxlabs/x/xy";
import "@/channel/LinePlot.css";

import { CSS } from "@synnaxlabs/charon/css";
import { Haul } from "@synnaxlabs/charon/haul";
import { usePrevious } from "@synnaxlabs/charon/hooks";
import type { Text } from "@synnaxlabs/charon/text";
import { type channel } from "@synnaxlabs/client";

import {
  type ReactElement,
  type Ref,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";

import { canDropHaulItem, filterHaulItems } from "@/channel/types";
import { LinePlot as Base } from "@/lineplot";
import { Range } from "@/lineplot/range";
import { Tooltip } from "@/lineplot/tooltip";
import { telem } from "@/telem/aether";
import { type Viewport } from "@/viewport";
import { Measure } from "@/vis/measure";
import { type measure } from "@/vis/measure/aether";
import { Rule } from "@/vis/rule";

/** Props for an axis in {@link LinePlot} */
export interface AxisProps extends Omit<Base.AxisProps, "axisKey"> {
  /** A unique identifier for the axis */
  key: string;
}

export interface BaseLineProps {
  key: string;
  axes: { x: string; y: string };
  channels: { y: channel.Key | channel.Name; x?: channel.Key | channel.Name };
  color: color.Crude;
  strokeWidth?: number;
  label?: string;
  downsample?: number;
  downsampleMode?: telem.DownsampleMode;
}

export interface StaticLineProps extends BaseLineProps {
  variant: "static";
  timeRange: xtelem.TimeRange;
}

export interface DynamicLineProps extends BaseLineProps {
  variant: "dynamic";
  timeSpan: xtelem.TimeSpan;
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

export interface LinePlotProps extends Omit<Base.LinePlotProps, "ref"> {
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
  legendVariant?: Base.LegendProps["variant"];
  legendPosition?: xy.XY;
  onLegendPositionChange?: (value: xy.XY) => void;
  enableTooltip?: boolean;
  enableMeasure?: boolean;
  initialViewport?: Viewport.UseProps["initial"];
  onViewportChange?: Viewport.UseProps["onChange"];
  viewportTriggers?: Viewport.UseProps["triggers"];
  rangeProviderProps?: Range.ProviderProps;
  measureMode?: measure.Mode;
  onMeasureModeChange?: (mode: measure.Mode) => void;
  ref?: Ref<Base.LinePlotRef>;
}

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
  rangeProviderProps,
  onSelectRule,
  measureMode,
  onMeasureModeChange,
  children,
  ref,
  ...rest
}: LinePlotProps): ReactElement => {
  const xAxes = axes.filter(({ location: l }) => loc.isY(l));
  const viewportRef = useRef<Viewport.UseRefValue | null>(null);
  const prevLinesLength = usePrevious(lines.length);
  const prevHold = usePrevious(rest.hold);
  const shouldResetViewport =
    (prevLinesLength === 0 && lines.length !== 0) ||
    (prevHold === true && rest.hold === false);
  useEffect(() => {
    if (shouldResetViewport) viewportRef.current?.reset();
  }, [shouldResetViewport]);
  return (
    <Base.LinePlot ref={ref} {...rest}>
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
            rangeProviderProps={rangeProviderProps}
            onRuleChange={onRuleChange}
            onSelectRule={onSelectRule}
          />
        );
      })}
      {showLegend && (
        <Base.Legend
          onLineChange={onLineChange}
          position={legendPosition}
          onPositionChange={onLegendPositionChange}
          variant={legendVariant}
        />
      )}
      {showTitle && (
        <Base.Title value={title} onChange={onTitleChange} level={titleLevel} />
      )}
      <Base.Viewport
        initial={initialViewport}
        onChange={onViewportChange}
        triggers={viewportTriggers}
        ref={viewportRef}
      >
        {enableTooltip && <Tooltip.Tooltip />}
        {enableMeasure && (
          <Measure.Measure mode={measureMode} onModeChange={onMeasureModeChange} />
        )}
        {children}
      </Base.Viewport>
    </Base.LinePlot>
  );
};

interface XAxisProps extends Pick<
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
  rangeProviderProps?: Range.ProviderProps;
}

export const useAxisDrop = (
  key: string,
  direction: direction.Direction,
  onAxisChannelDrop?: (key: string, keys: channel.Key[]) => void,
): Haul.UseDropReturn =>
  Haul.useDrop({
    type: `channel_lineplot_${direction}_axis`,
    canDrop: canDropHaulItem,
    onDrop: useCallback(
      ({ items }) => {
        const dropped = filterHaulItems(items);
        const keys = dropped.map(({ key }) => key);
        onAxisChannelDrop?.(key, keys);
        return dropped;
      },
      [key, onAxisChannelDrop],
    ),
  });

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
  rangeProviderProps,
}: XAxisProps): ReactElement => {
  const xRules = rules?.filter((r) => r.axis === key);
  const dragging = Haul.useDraggingState();
  const dropProps = useAxisDrop(key, "x", onAxisChannelDrop);
  return (
    <Base.XAxis
      {...axis}
      {...dropProps}
      location={location as loc.Y}
      axisKey={key}
      showGrid={showGrid ?? index === 0}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
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
      <Range.Provider {...rangeProviderProps} />
    </Base.XAxis>
  );
};

interface YAxisProps extends Pick<
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
  const dropProps = useAxisDrop(key, "y", onAxisChannelDrop);
  const dragging = Haul.useDraggingState();
  return (
    <Base.YAxis
      {...props}
      {...dropProps}
      location={loc as loc.X}
      axisKey={key}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
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
    </Base.YAxis>
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
    <Base.Line
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
    <Base.Line
      key={key}
      aetherKey={key}
      y={yTelem}
      x={xTelem}
      legendGroup={rest.axes.y.toUpperCase()}
      {...rest}
    />
  );
};

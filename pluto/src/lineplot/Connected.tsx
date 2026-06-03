// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot } from "@synnaxlabs/client";
import {
  box,
  color,
  DataType,
  type direction,
  location as loc,
  type optional,
  primitive,
  type TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";
import {
  type ReactElement,
  type ReactNode,
  type Ref,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { canDropHaulItem, filterHaulItems } from "@/channel/types";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useAsyncEffect, useDebouncedCallback, usePrevious } from "@/hooks";
import {
  type AxisProps as CoreAxisProps,
  XAxis as CoreXAxis,
  YAxis as CoreYAxis,
} from "@/lineplot/Axis";
import { axisLocation, isAxisKey, isXAxisKey, isYAxisKey } from "@/lineplot/axisKeys";
import { type DerivedLine, resolveLineColor, useDerivedLines } from "@/lineplot/derive";
import { Legend, type LegendProps } from "@/lineplot/Legend";
import { Line as CoreLine } from "@/lineplot/Line";
import { Core, type CoreProps, type CoreRef, type LineSpec } from "@/lineplot/LinePlot";
import { Measure } from "@/lineplot/measure";
import { type measure } from "@/lineplot/measure/aether";
import {
  useDispatch,
  useEnsureRetrieved,
  useRedo,
  useSelectAxes,
  useSelectChannels,
  useSelectLegend,
  useSelectLines,
  useSelectRanges,
  useSelectRules,
  useSelectTitle,
  useUndo,
} from "@/lineplot/queries";
import { Range } from "@/lineplot/range";
import { Rule } from "@/lineplot/rule";
import { Title } from "@/lineplot/Title";
import { Tooltip } from "@/lineplot/tooltip";
import { Viewport as CoreViewport } from "@/lineplot/Viewport";
import { Synnax } from "@/synnax";
import { telem } from "@/telem/aether";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";
import { type Viewport } from "@/viewport";

// A resolved range descriptor supplied by the consumer. Range resolution lives
// in the consumer (Console's range slice), so the connected component receives
// the static/dynamic time window per range key rather than reading it itself.
export type ResolvedRange =
  | { variant: "static"; timeRange: TimeRange }
  | { variant: "dynamic"; span: TimeSpan };

type AxisChange = Partial<CoreAxisProps> & { key: string };
type RuleChange = Partial<lineplot.Rule> & { key: string };

interface ResolvedLine {
  line: DerivedLine;
  color: color.Color;
  resolved?: ResolvedRange;
}

interface AxisRenderProps extends CoreAxisProps {
  key: string;
}

const useAxisDrop = (
  axisKey: string,
  direction: direction.Direction,
  onDrop?: (key: string, keys: channel.Key[]) => void,
): Haul.UseDropReturn =>
  Haul.useDrop({
    type: `channel_lineplot_${direction}_axis`,
    canDrop: canDropHaulItem,
    onDrop: useCallback(
      ({ items }) => {
        const dropped = filterHaulItems(items);
        onDrop?.(
          axisKey,
          dropped.map(({ key }) => key),
        );
        return dropped;
      },
      [axisKey, onDrop],
    ),
  });

const ConnectedLine = ({
  line,
  color,
  resolved,
}: ResolvedLine): ReactElement | null => {
  const telemetry = useMemo(() => {
    if (resolved == null) return null;
    const { xChannel, yChannel } = line;
    const hasX = xChannel != null && xChannel !== 0;
    if (resolved.variant === "dynamic") {
      const keepFor = Number(resolved.span.valueOf()) * 3;
      return {
        x: telem.streamChannelData({
          timeSpan: resolved.span,
          channel: hasX ? xChannel : yChannel,
          useIndexOfChannel: !hasX,
          keepFor,
        }),
        y: telem.streamChannelData({
          timeSpan: resolved.span,
          channel: yChannel,
          keepFor,
        }),
      };
    }
    return {
      x: telem.channelData({
        timeRange: resolved.timeRange,
        channel: hasX ? xChannel : yChannel,
        useIndexOfChannel: !hasX,
      }),
      y: telem.channelData({ timeRange: resolved.timeRange, channel: yChannel }),
    };
  }, [resolved, line.xChannel, line.yChannel]);
  if (telemetry == null) return null;
  return (
    <CoreLine
      aetherKey={line.key}
      x={telemetry.x}
      y={telemetry.y}
      color={color}
      strokeWidth={line.strokeWidth}
      label={line.label}
      downsample={line.downsample}
      downsampleMode={line.downsampleMode}
      legendGroup={line.yAxis.toUpperCase()}
    />
  );
};

interface AxisChildrenProps {
  lines: ResolvedLine[];
  rules: lineplot.Rule[];
  onRuleChange: (rule: RuleChange) => void;
  onSelectRule?: (key: string) => void;
}

const renderRules = (
  rules: lineplot.Rule[],
  axisKey: string,
  onRuleChange: (rule: RuleChange) => void,
  onSelectRule?: (key: string) => void,
): ReactElement[] =>
  rules
    .filter((r) => r.axis === axisKey)
    .map((r) => (
      <Rule.Rule
        key={r.key}
        aetherKey={r.key}
        position={r.position}
        color={r.color ?? color.ZERO}
        label={r.label}
        lineWidth={r.lineWidth}
        lineDash={r.lineDash}
        units={r.units}
        onLabelChange={(value) => onRuleChange({ key: r.key, label: value })}
        onPositionChange={(value) => onRuleChange({ key: r.key, position: value })}
        onUnitsChange={(value) => onRuleChange({ key: r.key, units: value })}
        onClick={() => onSelectRule?.(r.key)}
      />
    ));

interface YAxisProps extends AxisChildrenProps {
  axis: AxisRenderProps;
  onAxisChange: (a: AxisChange) => void;
  onChannelDrop?: (key: string, keys: channel.Key[]) => void;
}

const YAxis = ({
  axis: { location, key: axisKey, ...axis },
  lines,
  rules,
  onAxisChange,
  onChannelDrop,
  onRuleChange,
  onSelectRule,
}: YAxisProps): ReactElement => {
  const dropProps = useAxisDrop(axisKey, "y", onChannelDrop);
  const dragging = Haul.useDraggingState();
  return (
    <CoreYAxis
      {...axis}
      {...dropProps}
      location={location}
      axisKey={axisKey}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
      onLabelChange={(value) => onAxisChange({ key: axisKey, label: value })}
    >
      {lines
        .filter(({ line }) => line.yAxis === axisKey)
        .map((l) => (
          <ConnectedLine key={l.line.key} {...l} />
        ))}
      {renderRules(rules, axisKey, onRuleChange, onSelectRule)}
    </CoreYAxis>
  );
};

interface XAxisProps extends AxisChildrenProps {
  axis: AxisRenderProps;
  index: number;
  yAxes: AxisRenderProps[];
  onAxisChange: (a: AxisChange) => void;
  onChannelDrop?: (key: string, keys: channel.Key[]) => void;
  rangeProviderProps?: Range.ProviderProps;
}

const XAxis = ({
  axis: { location, key: axisKey, showGrid, ...axis },
  index,
  yAxes,
  lines,
  rules,
  onAxisChange,
  onChannelDrop,
  onRuleChange,
  onSelectRule,
  rangeProviderProps,
}: XAxisProps): ReactElement => {
  const dropProps = useAxisDrop(axisKey, "x", onChannelDrop);
  const dragging = Haul.useDraggingState();
  return (
    <CoreXAxis
      {...axis}
      {...dropProps}
      location={location}
      axisKey={axisKey}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
      showGrid={showGrid ?? index === 0}
      onLabelChange={(value) => onAxisChange({ key: axisKey, label: value })}
    >
      {yAxes.map((ya, j) => (
        <YAxis
          key={ya.key}
          axis={{ ...ya, showGrid: showGrid ?? (index === 0 && j === 0) }}
          lines={lines}
          rules={rules}
          onAxisChange={onAxisChange}
          onChannelDrop={onChannelDrop}
          onRuleChange={onRuleChange}
          onSelectRule={onSelectRule}
        />
      ))}
      {renderRules(rules, axisKey, onRuleChange, onSelectRule)}
      <Range.Provider {...rangeProviderProps} />
    </CoreXAxis>
  );
};

const shouldDisplayAxis = (
  key: lineplot.AxisKey,
  channels: lineplot.Channels,
): boolean => {
  if (key === "x1" || key === "y1") return true;
  if (key === "x2") return channels.x2 !== 0;
  return channels[key].length > 0;
};

const UNDO_REDO_CONFIG: Triggers.ModeConfig<"undo" | "redo" | "default"> = {
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  default: [],
  defaultMode: "default",
};
const UNDO_REDO_TRIGGERS = Triggers.flattenConfig(UNDO_REDO_CONFIG);

export interface LinePlotProps extends Omit<CoreProps, "ref"> {
  resourceKey: lineplot.Key;
  /** Gates every document mutation. */
  editable?: boolean;
  /** Gates the in-component undo/redo keyboard shortcuts; the consumer scopes
   * this to the focused tab and the user's update permission. */
  enableTriggers?: boolean | (() => boolean);
  /** Range key -> resolved static/dynamic window, supplied by the consumer. */
  resolvedRanges?: Map<string, ResolvedRange>;
  /** Active range key prepended when dropping a channel onto an empty plot. */
  activeRangeKey?: string;
  title?: string;
  onTitleChange?: (value: string) => void;
  legendVariant?: LegendProps["variant"];
  enableTooltip?: boolean;
  enableMeasure?: boolean;
  measureMode?: measure.Mode;
  onMeasureModeChange?: (mode: measure.Mode) => void;
  initialViewport?: Viewport.UseProps["initial"];
  onViewportChange?: Viewport.UseProps["onChange"];
  viewportTriggers?: Viewport.UseProps["triggers"];
  rangeProviderProps?: Range.ProviderProps;
  onSelectRule?: (key: string) => void;
  children?: ReactNode;
  ref?: Ref<CoreRef>;
}

export const LinePlot = ({
  resourceKey: key,
  editable = false,
  enableTriggers = true,
  resolvedRanges,
  activeRangeKey,
  title,
  onTitleChange,
  legendVariant,
  enableTooltip = true,
  enableMeasure = false,
  measureMode,
  onMeasureModeChange,
  initialViewport = box.DECIMAL,
  onViewportChange,
  viewportTriggers,
  rangeProviderProps,
  onSelectRule,
  children,
  ref,
  ...rest
}: LinePlotProps): ReactElement => {
  useEnsureRetrieved({ key });
  const theme = Theming.use();
  const client = Synnax.use();
  const { dispatch } = useDispatch();
  const { undo } = useUndo({ key });
  const { redo } = useRedo({ key });
  Triggers.use({
    triggers: UNDO_REDO_TRIGGERS,
    loose: true,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        if (enableTriggers === false) return;
        if (typeof enableTriggers === "function" && !enableTriggers()) return;
        const mode = Triggers.determineMode(UNDO_REDO_CONFIG, triggers);
        if (mode === "undo") undo();
        else if (mode === "redo") redo();
      },
      [enableTriggers, undo, redo],
    ),
  });
  const titleState = useSelectTitle({ key });
  const legend = useSelectLegend({ key });
  const channels = useSelectChannels({ key });
  const ranges = useSelectRanges({ key });
  const rules = useSelectRules({ key });
  const axes = useSelectAxes({ key });
  const storedLines = useSelectLines({ key });
  const derived = useDerivedLines(key);

  const palette = theme.colors.visualization.palettes.default;

  useAsyncEffect(
    async (signal) => {
      if (!editable || client == null) return;
      const stored = new Set(storedLines.map((l) => l.key));
      const toMaterialize = derived.filter(
        (l) => l.label == null && !stored.has(l.key),
      );
      if (toMaterialize.length === 0) return;
      const fetched = await client.channels.retrieve(
        toMaterialize.map((l) => l.yChannel),
      );
      if (signal.aborted) return;
      dispatch({
        key,
        actions: toMaterialize.map((l) =>
          lineplot.setLine({
            line: {
              ...l,
              label: fetched.find((f) => f.key === l.yChannel)?.name ?? undefined,
              color: resolveLineColor(l.color, derived.indexOf(l), palette),
            },
          }),
        ),
      });
    },
    [key, client, derived, storedLines, editable, palette],
  );

  useAsyncEffect(async () => {
    if (!editable || client == null) return;
    const ax = axes.x1;
    const ch = channels.x1;
    let newType: lineplot.TickType = "time";
    if (primitive.isNonZero(ch)) {
      const retrieved = await client.channels.retrieve(ch);
      if (!retrieved.dataType.equals(DataType.TIMESTAMP)) newType = "linear";
    }
    if (ax.type === newType) return;
    dispatch({ key, actions: [lineplot.setAxis({ axis: { ...ax, type: newType } })] });
  }, [channels.x1, axes.x1.type, client, editable]);

  const handleLineChange = useCallback(
    (d: optional.Optional<LineSpec, "legendGroup">) => {
      const existing =
        storedLines.find((l) => l.key === d.key) ??
        derived.find((l) => l.key === d.key);
      if (existing == null) return;
      const next: lineplot.Line = {
        ...existing,
        label: d.label,
        color: color.construct(d.color),
      };
      dispatch({ key, actions: [lineplot.setLine({ line: next })] });
    },
    [dispatch, key, storedLines, derived],
  );

  const handleRuleChange = useCallback(
    (rule: RuleChange) => {
      const existing = rules.find((r) => r.key === rule.key);
      if (existing == null) return;
      const nextAxis =
        rule.axis != null && isAxisKey(rule.axis) ? rule.axis : existing.axis;
      const next: lineplot.Rule = {
        ...existing,
        ...rule,
        color: rule.color != null ? color.construct(rule.color) : existing.color,
        axis: nextAxis,
      };
      dispatch({ key, actions: [lineplot.setRule({ rule: next })] });
    },
    [dispatch, key, rules],
  );

  const handleAxisChange = useCallback(
    (a: AxisChange) => {
      if (!isAxisKey(a.key)) return;
      const existing = axes[a.key];
      if (existing == null || a.label == null) return;
      const next: lineplot.Axis = { ...existing, key: a.key, label: a.label };
      dispatch({ key, actions: [lineplot.setAxis({ axis: next })] });
    },
    [dispatch, key, axes],
  );

  const handleChannelDrop = useCallback(
    (axisKey: string, dropped: channel.Key[]): void => {
      const actions: lineplot.Action[] = [];
      if (isXAxisKey(axisKey))
        actions.push(lineplot.setXChannel({ axisKey, channel: dropped[0] }));
      else if (isYAxisKey(axisKey)) {
        const existing = new Set(channels[axisKey]);
        for (const c of dropped)
          if (!existing.has(c))
            actions.push(lineplot.addChannel({ axisKey, channel: c }));
      } else return;
      if (
        activeRangeKey != null &&
        ranges.x1.length === 0 &&
        ranges.x2.length === 0 &&
        actions.length > 0
      )
        actions.unshift(lineplot.addRange({ axisKey: "x1", range: activeRangeKey }));
      dispatch({ key, actions });
    },
    [dispatch, key, channels, ranges, activeRangeKey],
  );

  const [legendPosition, setLegendPosition] = useState(legend.position);
  const storeLegendPosition = useDebouncedCallback(
    (position: typeof legend.position) =>
      dispatch({
        key,
        actions: [lineplot.setLegend({ legend: { ...legend, position } })],
      }),
    TimeSpan.milliseconds(100),
    [dispatch, key, legend],
  );
  const handleLegendPositionChange = useCallback(
    (position: typeof legend.position) => {
      setLegendPosition(position);
      storeLegendPosition(position);
    },
    [storeLegendPosition],
  );

  const builtAxes = useMemo<AxisRenderProps[]>(
    () =>
      lineplot.AXIS_KEYS.filter((k) => shouldDisplayAxis(k, channels)).map(
        (k): AxisRenderProps => ({ ...axes[k], location: axisLocation(k), key: k }),
      ),
    [axes, channels],
  );

  const resolvedLines = useMemo<ResolvedLine[]>(
    () =>
      derived.map((line, i) => ({
        line,
        color: resolveLineColor(line.color, i, palette),
        resolved: resolvedRanges?.get(line.range),
      })),
    [derived, palette, resolvedRanges],
  );

  const viewportRef = useRef<Viewport.UseRefValue | null>(null);
  const prevLen = usePrevious(derived.length);
  const prevHold = usePrevious(rest.hold);
  useEffect(() => {
    if (
      (prevLen === 0 && derived.length !== 0) ||
      (prevHold === true && rest.hold === false)
    )
      viewportRef.current?.reset();
  }, [derived.length, rest.hold]);

  const xAxes = builtAxes.filter(({ location }) => loc.isY(location));
  const yAxes = builtAxes.filter(({ location }) => loc.isX(location));

  return (
    <Core ref={ref} {...rest}>
      {xAxes.map((axis, i) => (
        <XAxis
          key={axis.key}
          axis={axis}
          index={i}
          yAxes={yAxes}
          lines={resolvedLines}
          rules={rules}
          onAxisChange={handleAxisChange}
          onChannelDrop={editable ? handleChannelDrop : undefined}
          onRuleChange={handleRuleChange}
          onSelectRule={onSelectRule}
          rangeProviderProps={rangeProviderProps}
        />
      ))}
      {legend.visible && (
        <Legend
          onLineChange={editable ? handleLineChange : undefined}
          position={legendPosition}
          onPositionChange={editable ? handleLegendPositionChange : undefined}
          variant={legendVariant}
        />
      )}
      {titleState.visible && (
        <Title
          value={title ?? ""}
          onChange={editable ? onTitleChange : undefined}
          level={titleState.level}
        />
      )}
      <CoreViewport
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
      </CoreViewport>
    </Core>
  );
};

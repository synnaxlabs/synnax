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
  type direction,
  type location,
  type optional,
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
import { useDebouncedCallback, usePrevious } from "@/hooks";
import {
  type AxisProps as CoreAxisProps,
  XAxis as CoreXAxis,
  YAxis as CoreYAxis,
} from "@/lineplot/Axis";
import { Frame, type FrameProps, type FrameRef, type LineSpec } from "@/lineplot/Frame";
import { Legend, type LegendProps } from "@/lineplot/Legend";
import { Line as CoreLine } from "@/lineplot/Line";
import { Measure } from "@/lineplot/measure";
import { type measure } from "@/lineplot/measure/aether";
import {
  useDispatch,
  useEnsureRetrieved,
  useRedo,
  useSelectAxisRuleKeys,
  useSelectChannels,
  useSelectLegend,
  useSelectLine,
  useSelectLineKeys,
  useSelectRanges,
  useSelectRule,
  useSelectTitle,
  useSelectXAxis,
  useSelectYAxis,
  useUndo,
} from "@/lineplot/queries";
import { Range } from "@/lineplot/range";
import { Rule } from "@/lineplot/rule";
import { Title } from "@/lineplot/Title";
import { Tooltip } from "@/lineplot/tooltip";
import { Viewport as CoreViewport } from "@/lineplot/Viewport";
import { telem } from "@/telem/aether";
import { Triggers } from "@/triggers";
import { type Viewport } from "@/viewport";

// A resolved range descriptor supplied by the consumer. Range resolution lives
// in the consumer (Console's range slice), so the connected component receives
// the static/dynamic time window per range key rather than reading it itself.
export type ResolvedRange =
  | { variant: "static"; timeRange: TimeRange }
  | { variant: "dynamic"; span: TimeSpan };

export const axisLabel = (key: lineplot.AxisKey): string => key.toUpperCase();

const AXIS_LOCATIONS: Record<lineplot.AxisKey, location.Outer> = {
  y1: "left",
  y2: "right",
  y3: "left",
  y4: "right",
  x1: "bottom",
  x2: "top",
};

type AxisChange = Partial<CoreAxisProps> & { key: lineplot.AxisKey };
type RuleChange = Partial<lineplot.Rule> & { key: string };

const useAxisDrop = <K extends lineplot.AxisKey>(
  axisKey: K,
  direction: direction.Direction,
  onDrop?: (key: K, keys: channel.Key[]) => void,
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

interface ConnectedLineProps {
  pKey: lineplot.Key;
  lineKey: string;
  resolved?: ResolvedRange;
}

const ConnectedLine = ({
  pKey,
  lineKey,
  resolved,
}: ConnectedLineProps): ReactElement | null => {
  const line = useSelectLine({ key: pKey, lineKey });
  const telemetry = useMemo(() => {
    if (line == null || resolved == null) return null;
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
  }, [resolved, line?.xChannel, line?.yChannel]);
  if (line == null || telemetry == null) return null;
  return (
    <CoreLine
      aetherKey={line.key}
      x={telemetry.x}
      y={telemetry.y}
      color={line.color}
      strokeWidth={line.strokeWidth}
      label={line.label}
      downsample={line.downsample}
      downsampleMode={line.downsampleMode}
      legendGroup={line.yAxis.toUpperCase()}
    />
  );
};

interface ConnectedTitleProps {
  pKey: lineplot.Key;
  value: string;
  onChange?: (value: string) => void;
}

const ConnectedTitle = ({
  pKey,
  value,
  onChange,
}: ConnectedTitleProps): ReactElement | null => {
  const { visible, level } = useSelectTitle({ key: pKey });
  if (!visible) return null;
  return <Title value={value} onChange={onChange} level={level} />;
};

interface ConnectedLegendProps {
  pKey: lineplot.Key;
  variant?: LegendProps["variant"];
  onLineChange?: LegendProps["onLineChange"];
  editable?: boolean;
}

const ConnectedLegend = ({
  pKey,
  variant,
  onLineChange,
  editable,
}: ConnectedLegendProps): ReactElement | null => {
  const { dispatch } = useDispatch();
  const legend = useSelectLegend({ key: pKey });
  // Position is held locally for responsive dragging and debounced into the
  // store so the document isn't written on every pointer move.
  const [position, setPosition] = useState(legend.position);
  const storePosition = useDebouncedCallback(
    (next: typeof legend.position) =>
      dispatch({
        key: pKey,
        actions: [lineplot.setLegend({ legend: { ...legend, position: next } })],
      }),
    TimeSpan.milliseconds(100),
    [dispatch, pKey, legend],
  );
  const handlePositionChange = useCallback(
    (next: typeof legend.position) => {
      setPosition(next);
      storePosition(next);
    },
    [storePosition],
  );
  if (!legend.visible) return null;
  return (
    <Legend
      onLineChange={onLineChange}
      position={position}
      onPositionChange={editable ? handlePositionChange : undefined}
      variant={variant}
    />
  );
};

interface AxisChildrenProps {
  pKey: lineplot.Key;
  resolvedRanges?: Map<string, ResolvedRange>;
  onSelectRule?: (key: string) => void;
}

interface ConnectedRuleProps {
  pKey: lineplot.Key;
  ruleKey: string;
  onSelectRule?: (key: string) => void;
}

const ConnectedRule = ({
  pKey,
  ruleKey,
  onSelectRule,
}: ConnectedRuleProps): ReactElement | null => {
  const { dispatch } = useDispatch();
  const rule = useSelectRule({ key: pKey, ruleKey });
  const update = useCallback(
    (next: RuleChange) => {
      if (rule == null) return;
      dispatch({
        key: pKey,
        actions: [
          lineplot.setRule({
            rule: {
              ...rule,
              ...next,
              color: next.color != null ? color.construct(next.color) : rule.color,
              axis: next.axis ?? rule.axis,
            },
          }),
        ],
      });
    },
    [dispatch, pKey, rule],
  );
  if (rule == null) return null;
  return (
    <Rule.Rule
      aetherKey={rule.key}
      position={rule.position}
      color={rule.color ?? color.ZERO}
      label={rule.label}
      lineWidth={rule.lineWidth}
      lineDash={rule.lineDash}
      units={rule.units}
      onLabelChange={(value) => update({ key: rule.key, label: value })}
      onPositionChange={(value) => update({ key: rule.key, position: value })}
      onUnitsChange={(value) => update({ key: rule.key, units: value })}
      onClick={() => onSelectRule?.(rule.key)}
    />
  );
};

interface RulesProps {
  pKey: lineplot.Key;
  axisKey: lineplot.AxisKey;
  onSelectRule?: (key: string) => void;
}

const Rules = ({ pKey, axisKey, onSelectRule }: RulesProps): ReactElement => {
  const ruleKeys = useSelectAxisRuleKeys({ key: pKey, axisKey });
  return (
    <>
      {ruleKeys.map((ruleKey) => (
        <ConnectedRule
          key={ruleKey}
          pKey={pKey}
          ruleKey={ruleKey}
          onSelectRule={onSelectRule}
        />
      ))}
    </>
  );
};

interface YAxisProps extends AxisChildrenProps {
  axisKey: lineplot.YAxisKey;
  onAxisChange: (a: AxisChange) => void;
  onChannelDrop?: (key: lineplot.YAxisKey, keys: channel.Key[]) => void;
}

const YAxis = ({
  pKey,
  axisKey,
  resolvedRanges,
  onAxisChange,
  onChannelDrop,
  onSelectRule,
}: YAxisProps): ReactElement => {
  const dropProps = useAxisDrop(axisKey, "y", onChannelDrop);
  const dragging = Haul.useDraggingState();
  const { axis, lineKeys } = useSelectYAxis({ key: pKey, axisKey });
  const { key: _axisKey, ...axisConfig } = axis;
  return (
    <CoreYAxis
      {...axisConfig}
      {...dropProps}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      showGrid={axisKey === "y1"}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
      onLabelChange={(value) => onAxisChange({ key: axisKey, label: value })}
    >
      {lineKeys.map((lineKey) => (
        <ConnectedLine
          key={lineKey}
          pKey={pKey}
          lineKey={lineKey}
          resolved={resolvedRanges?.get(lineplot.parseLineKey(lineKey).range)}
        />
      ))}
      <Rules pKey={pKey} axisKey={axisKey} onSelectRule={onSelectRule} />
    </CoreYAxis>
  );
};

interface XAxisProps extends AxisChildrenProps {
  axisKey: lineplot.XAxisKey;
  yAxes: lineplot.YAxisKey[];
  onAxisChange: (a: AxisChange) => void;
  onXChannelDrop?: (key: lineplot.XAxisKey, keys: channel.Key[]) => void;
  onYChannelDrop?: (key: lineplot.YAxisKey, keys: channel.Key[]) => void;
  rangeProviderProps?: Range.ProviderProps;
}

const XAxis = ({
  pKey,
  axisKey,
  yAxes,
  resolvedRanges,
  onAxisChange,
  onXChannelDrop,
  onYChannelDrop,
  onSelectRule,
  rangeProviderProps,
}: XAxisProps): ReactElement => {
  const dropProps = useAxisDrop(axisKey, "x", onXChannelDrop);
  const dragging = Haul.useDraggingState();
  const { key: _axisKey, ...axisConfig } = useSelectXAxis({ key: pKey, axisKey });
  return (
    <CoreXAxis
      {...axisConfig}
      {...dropProps}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
      showGrid={axisKey === "x1"}
      onLabelChange={(value) => onAxisChange({ key: axisKey, label: value })}
    >
      {yAxes.map((yAxisKey) => (
        <YAxis
          key={yAxisKey}
          pKey={pKey}
          axisKey={yAxisKey}
          resolvedRanges={resolvedRanges}
          onAxisChange={onAxisChange}
          onChannelDrop={onYChannelDrop}
          onSelectRule={onSelectRule}
        />
      ))}
      <Rules pKey={pKey} axisKey={axisKey} onSelectRule={onSelectRule} />
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

export interface LinePlotProps extends Omit<FrameProps, "ref"> {
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
  ref?: Ref<FrameRef>;
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
  const channels = useSelectChannels({ key });
  const ranges = useSelectRanges({ key });
  const lineKeys = useSelectLineKeys({ key });

  const handleLineChange = useCallback(
    (d: optional.Optional<LineSpec, "legendGroup">) => {
      dispatch({
        key,
        actions: [
          lineplot.setLine({
            key: d.key,
            label: d.label,
            color: color.construct(d.color),
          }),
        ],
      });
    },
    [dispatch, key],
  );

  const handleAxisChange = useCallback(
    (a: AxisChange) => {
      if (a.label == null) return;
      dispatch({ key, actions: [lineplot.setAxis({ key: a.key, label: a.label })] });
    },
    [dispatch, key],
  );

  const dispatchChannelDrop = useCallback(
    (actions: lineplot.Action[]): void => {
      if (
        activeRangeKey != null &&
        ranges.x1.length === 0 &&
        ranges.x2.length === 0 &&
        actions.length > 0
      )
        actions.unshift(lineplot.addRange({ axisKey: "x1", range: activeRangeKey }));
      dispatch({ key, actions });
    },
    [dispatch, key, ranges, activeRangeKey],
  );

  const handleXChannelDrop = useCallback(
    (axisKey: lineplot.XAxisKey, dropped: channel.Key[]): void => {
      dispatchChannelDrop([lineplot.setXChannel({ axisKey, channel: dropped[0] })]);
    },
    [dispatchChannelDrop],
  );

  const handleYChannelDrop = useCallback(
    (axisKey: lineplot.YAxisKey, dropped: channel.Key[]): void => {
      const existing = new Set(channels[axisKey]);
      const actions: lineplot.Action[] = [];
      for (const c of dropped)
        if (!existing.has(c))
          actions.push(lineplot.addChannel({ axisKey, channel: c }));
      dispatchChannelDrop(actions);
    },
    [channels, dispatchChannelDrop],
  );

  const xAxisKeys = useMemo(
    () => lineplot.X_AXIS_KEYS.filter((k) => shouldDisplayAxis(k, channels)),
    [channels],
  );
  const yAxisKeys = useMemo(
    () => lineplot.Y_AXIS_KEYS.filter((k) => shouldDisplayAxis(k, channels)),
    [channels],
  );

  const viewportRef = useRef<Viewport.UseRefValue | null>(null);
  const prevLen = usePrevious(lineKeys.length);
  const prevHold = usePrevious(rest.hold);
  useEffect(() => {
    if (
      (prevLen === 0 && lineKeys.length !== 0) ||
      (prevHold === true && rest.hold === false)
    )
      viewportRef.current?.reset();
  }, [lineKeys.length, rest.hold]);

  return (
    <Frame ref={ref} {...rest}>
      {xAxisKeys.map((xAxisKey) => (
        <XAxis
          key={xAxisKey}
          pKey={key}
          axisKey={xAxisKey}
          yAxes={yAxisKeys}
          resolvedRanges={resolvedRanges}
          onAxisChange={handleAxisChange}
          onXChannelDrop={editable ? handleXChannelDrop : undefined}
          onYChannelDrop={editable ? handleYChannelDrop : undefined}
          onSelectRule={onSelectRule}
          rangeProviderProps={rangeProviderProps}
        />
      ))}
      <ConnectedLegend
        pKey={key}
        variant={legendVariant}
        onLineChange={editable ? handleLineChange : undefined}
        editable={editable}
      />
      <ConnectedTitle
        pKey={key}
        value={title ?? ""}
        onChange={editable ? onTitleChange : undefined}
      />
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
    </Frame>
  );
};

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
  type TimeSpan,
} from "@synnaxlabs/x";
import {
  type ReactElement,
  type RefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";

import { canDropHaulItem, filterHaulItems } from "@/channel/types";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { usePrevious } from "@/hooks";
import { XAxis as BaseXAxis, YAxis as BaseYAxis } from "@/lineplot/Axis";
import { Frame, type FrameProps, type LineSpec } from "@/lineplot/Frame";
import {
  Legend as BaseLegend,
  type LegendProps as BaseLegendProps,
} from "@/lineplot/Legend";
import { Line as BaseLine } from "@/lineplot/Line";
import { Measure } from "@/lineplot/measure";
import { type measure } from "@/lineplot/measure/aether";
import {
  useDispatch,
  useEnsureRetrieved,
  useRedo,
  useRename,
  useSelectAxisRuleKeys,
  useSelectChannels,
  useSelectLegend,
  useSelectLine,
  useSelectLineCount,
  useSelectName,
  useSelectRanges,
  useSelectRule,
  useSelectTitle,
  useSelectXAxis,
  useSelectXAxisKeys,
  useSelectYAxis,
  useSelectYAxisKeys,
  useUndo,
} from "@/lineplot/queries";
import { Range } from "@/lineplot/range";
import { Rule as BaseRule } from "@/lineplot/rule";
import { Title as BaseTitle } from "@/lineplot/Title";
import { Tooltip } from "@/lineplot/tooltip";
import { Viewport as BaseViewport } from "@/lineplot/Viewport";
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

type RuleChange = Partial<lineplot.Rule> & { key: string };

const UNDO_REDO_CONFIG: Triggers.ModeConfig<"undo" | "redo" | "default"> = {
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  default: [],
  defaultMode: "default",
};
const UNDO_REDO_TRIGGERS = Triggers.flattenConfig(UNDO_REDO_CONFIG);

interface UseUndoRedoTriggersProps {
  key: lineplot.Key;
  enabled: boolean | (() => boolean);
}

const useUndoRedoTriggers = ({ key, enabled }: UseUndoRedoTriggersProps) => {
  const { undo } = useUndo({ key });
  const { redo } = useRedo({ key });
  Triggers.use({
    triggers: UNDO_REDO_TRIGGERS,
    loose: true,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        if (enabled === false) return;
        if (typeof enabled === "function" && !enabled()) return;
        const mode = Triggers.determineMode(UNDO_REDO_CONFIG, triggers);
        if (mode === "undo") undo();
        else if (mode === "redo") redo();
      },
      [enabled, undo, redo],
    ),
  });
};

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

// useChannelDropper commits the actions produced by a channel drop. If the plot
// has no ranges yet and an active range is provided, it first binds that range
// to x1 so the dropped channel has something to plot against.
const useChannelDropper = (
  pKey: lineplot.Key,
  activeRangeKey?: string,
): ((actions: lineplot.Action[]) => void) => {
  const { dispatch } = useDispatch();
  const ranges = useSelectRanges({ key: pKey });
  return useCallback(
    (actions: lineplot.Action[]): void => {
      if (
        activeRangeKey != null &&
        ranges.x1.length === 0 &&
        ranges.x2.length === 0 &&
        actions.length > 0
      )
        actions.unshift(lineplot.addRange({ axisKey: "x1", range: activeRangeKey }));
      dispatch({ key: pKey, actions });
    },
    [dispatch, pKey, ranges, activeRangeKey],
  );
};

interface LineProps {
  pKey: lineplot.Key;
  lineKey: string;
  resolved?: ResolvedRange;
}

const Line = ({ pKey, lineKey, resolved }: LineProps): ReactElement | null => {
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
    <BaseLine
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

interface TitleProps {
  pKey: lineplot.Key;
  editable?: boolean;
}

const Title = ({ pKey, editable }: TitleProps): ReactElement | null => {
  const { visible, level } = useSelectTitle({ key: pKey });
  const name = useSelectName({ key: pKey });
  const { update: rename } = useRename({});
  const handleChange = useCallback(
    (value: string) => {
      rename({ key: pKey, name: value });
    },
    [rename, pKey],
  );
  if (!visible) return null;
  return (
    <BaseTitle
      value={name}
      onChange={handleChange}
      disabled={editable !== true}
      level={level}
    />
  );
};

interface LegendProps {
  pKey: lineplot.Key;
  variant?: BaseLegendProps["variant"];
  editable?: boolean;
}

const Legend = ({ pKey, variant, editable }: LegendProps): ReactElement | null => {
  const { dispatch } = useDispatch();
  const legend = useSelectLegend({ key: pKey });
  const handlePositionChange = useCallback(
    (position: typeof legend.position) =>
      dispatch({ key: pKey, actions: [lineplot.setLegendPosition({ position })] }),
    [dispatch, pKey],
  );
  const handleLineChange = useCallback(
    (d: optional.Optional<LineSpec, "legendGroup">) => {
      dispatch({
        key: pKey,
        actions: [
          lineplot.setLineLabel({ key: d.key, label: d.label }),
          lineplot.setLineColor({ key: d.key, color: color.construct(d.color) }),
        ],
      });
    },
    [dispatch, pKey],
  );
  if (!legend.visible) return null;
  // The legend has no disabled prop; absent callbacks are how it disables drag
  // and entry editing, so gate at the prop rather than inside the handlers.
  return (
    <BaseLegend
      onLineChange={editable ? handleLineChange : undefined}
      position={legend.position}
      onPositionChange={editable ? handlePositionChange : undefined}
      variant={variant}
    />
  );
};

interface AxisChildrenProps {
  pKey: lineplot.Key;
  editable?: boolean;
  activeRangeKey?: string;
  resolvedRanges?: Map<string, ResolvedRange>;
  onSelectRule?: (key: string) => void;
}

interface RuleProps {
  pKey: lineplot.Key;
  ruleKey: string;
  onSelectRule?: (key: string) => void;
}

const Rule = ({ pKey, ruleKey, onSelectRule }: RuleProps): ReactElement | null => {
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
    <BaseRule.Rule
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
        <Rule key={ruleKey} pKey={pKey} ruleKey={ruleKey} onSelectRule={onSelectRule} />
      ))}
    </>
  );
};

interface YAxisProps extends AxisChildrenProps {
  axisKey: lineplot.YAxisKey;
}

const YAxis = ({
  pKey,
  axisKey,
  editable,
  activeRangeKey,
  resolvedRanges,
  onSelectRule,
}: YAxisProps): ReactElement => {
  const { dispatch } = useDispatch();
  const channels = useSelectChannels({ key: pKey });
  const commitDrop = useChannelDropper(pKey, activeRangeKey);
  const handleDrop = useCallback(
    (k: lineplot.YAxisKey, dropped: channel.Key[]): void => {
      if (editable !== true) return;
      const existing = new Set(channels[k]);
      const actions: lineplot.Action[] = [];
      for (const c of dropped)
        if (!existing.has(c))
          actions.push(lineplot.addChannel({ axisKey: k, channel: c }));
      commitDrop(actions);
    },
    [channels, commitDrop, editable],
  );
  const dropProps = useAxisDrop(axisKey, "y", handleDrop);
  const dragging = Haul.useDraggingState();
  const { axis, lineKeys } = useSelectYAxis({ key: pKey, axisKey });
  const { key: _axisKey, ...axisConfig } = axis;
  return (
    <BaseYAxis
      {...axisConfig}
      {...dropProps}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      showGrid={axisKey === "y1"}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
      onLabelChange={(value) =>
        dispatch({
          key: pKey,
          actions: [lineplot.setAxisLabel({ key: axisKey, label: value })],
        })
      }
    >
      {lineKeys.map((lineKey) => (
        <Line
          key={lineKey}
          pKey={pKey}
          lineKey={lineKey}
          resolved={resolvedRanges?.get(lineplot.parseLineKey(lineKey).range)}
        />
      ))}
      <Rules pKey={pKey} axisKey={axisKey} onSelectRule={onSelectRule} />
    </BaseYAxis>
  );
};

interface XAxisProps extends AxisChildrenProps {
  axisKey: lineplot.XAxisKey;
  rangeProviderProps?: Range.ProviderProps;
}

const XAxis = ({
  pKey,
  axisKey,
  editable,
  activeRangeKey,
  resolvedRanges,
  onSelectRule,
  rangeProviderProps,
}: XAxisProps): ReactElement => {
  const { dispatch } = useDispatch();
  const commitDrop = useChannelDropper(pKey, activeRangeKey);
  const handleDrop = useCallback(
    (k: lineplot.XAxisKey, dropped: channel.Key[]): void => {
      if (editable !== true) return;
      commitDrop([lineplot.setXChannel({ axisKey: k, channel: dropped[0] })]);
    },
    [commitDrop, editable],
  );
  const dropProps = useAxisDrop(axisKey, "x", handleDrop);
  const dragging = Haul.useDraggingState();
  const { key: _, ...axisConfig } = useSelectXAxis({ key: pKey, axisKey });
  const yAxes = useSelectYAxisKeys({ key: pKey });
  const handleLabelChange = useCallback(
    (label: string) =>
      dispatch({
        key: pKey,
        actions: [lineplot.setAxisLabel({ key: axisKey, label })],
      }),
    [dispatch, pKey, axisKey],
  );
  return (
    <BaseXAxis
      {...axisConfig}
      {...dropProps}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
      showGrid={axisKey === "x1"}
      onLabelChange={handleLabelChange}
    >
      {yAxes.map((yAxisKey) => (
        <YAxis
          key={yAxisKey}
          pKey={pKey}
          axisKey={yAxisKey}
          editable={editable}
          activeRangeKey={activeRangeKey}
          resolvedRanges={resolvedRanges}
          onSelectRule={onSelectRule}
        />
      ))}
      <Rules pKey={pKey} axisKey={axisKey} onSelectRule={onSelectRule} />
      <Range.Provider {...rangeProviderProps} />
    </BaseXAxis>
  );
};

interface UseViewportResetParams {
  key: lineplot.Key;
  hold?: boolean;
}

const useViewportReset = ({
  key,
  hold,
}: UseViewportResetParams): RefObject<Viewport.UseRefValue | null> => {
  const lineCount = useSelectLineCount({ key });
  const prevLineCount = usePrevious(lineCount);
  const prevHold = usePrevious(hold);
  const viewportRef = useRef<Viewport.UseRefValue | null>(null);
  useEffect(() => {
    if (
      (prevLineCount === 0 && lineCount !== 0) ||
      (prevHold === true && hold === false)
    )
      viewportRef.current?.reset();
  }, [lineCount, hold]);
  return viewportRef;
};

export interface LinePlotProps extends FrameProps {
  resourceKey: lineplot.Key;
  editable?: boolean;
  enableTriggers?: boolean | (() => boolean);
  resolvedRanges?: Map<string, ResolvedRange>;
  activeRangeKey?: string;
  legendVariant?: BaseLegendProps["variant"];
  enableTooltip?: boolean;
  enableMeasure?: boolean;
  measureMode?: measure.Mode;
  onMeasureModeChange?: (mode: measure.Mode) => void;
  initialViewport?: Viewport.UseProps["initial"];
  onViewportChange?: Viewport.UseProps["onChange"];
  viewportTriggers?: Viewport.UseProps["triggers"];
  rangeProviderProps?: Range.ProviderProps;
  onSelectRule?: (key: string) => void;
}

export const LinePlot = ({
  resourceKey: key,
  editable = false,
  enableTriggers = true,
  resolvedRanges,
  activeRangeKey,
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
  useUndoRedoTriggers({ key, enabled: enableTriggers });
  const xAxisKeys = useSelectXAxisKeys({ key });
  const viewportRef = useViewportReset({ key, hold: rest.hold });
  return (
    <Frame ref={ref} {...rest}>
      {xAxisKeys.map((xAxisKey) => (
        <XAxis
          key={xAxisKey}
          pKey={key}
          axisKey={xAxisKey}
          editable={editable}
          activeRangeKey={activeRangeKey}
          resolvedRanges={resolvedRanges}
          onSelectRule={onSelectRule}
          rangeProviderProps={rangeProviderProps}
        />
      ))}
      <Legend pKey={key} variant={legendVariant} editable={editable} />
      <Title pKey={key} editable={editable} />
      <BaseViewport
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
      </BaseViewport>
    </Frame>
  );
};

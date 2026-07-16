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
  type color,
  type direction,
  type location,
  primitive,
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
import { Frame, type FrameProps } from "@/lineplot/Frame";
import {
  Legend as BaseLegend,
  type LegendProps as BaseLegendProps,
} from "@/lineplot/Legend";
import { Line as BaseLine } from "@/lineplot/Line";
import { Measure } from "@/lineplot/measure";
import { type measure } from "@/lineplot/measure/aether";
import {
  useDispatch,
  useRedo,
  useRename,
  useSelectAxisRuleKeys,
  useSelectLegend,
  useSelectLine,
  useSelectLineCount,
  useSelectName,
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
import { useKey } from "@/lineplot/Suspended";
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
  { variant: "static"; timeRange: TimeRange } | { variant: "dynamic"; span: TimeSpan };

export const axisLabel = (key: lineplot.AxisKey): string => key.toUpperCase();

const AXIS_LOCATIONS: Record<lineplot.AxisKey, location.Outer> = {
  y1: "left",
  y2: "right",
  y3: "left",
  y4: "right",
  x1: "bottom",
  x2: "top",
};

const UNDO_REDO_CONFIG: Triggers.ModeConfig<"undo" | "redo" | "default"> = {
  undo: [Triggers.UNDO],
  redo: [Triggers.REDO],
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
        const keys = dropped.map(({ key }) => key);
        onDrop?.(axisKey, keys);
        return dropped;
      },
      [axisKey, onDrop],
    ),
  });

interface LineProps {
  pKey: lineplot.Key;
  lineKey: string;
  resolved?: ResolvedRange;
  visible?: boolean;
}

const Line = ({
  pKey,
  lineKey,
  resolved,
  visible = true,
}: LineProps): ReactElement | null => {
  const { key, ...line } = useSelectLine({ key: pKey, lineKey });
  const telemetry = useMemo(() => {
    if (resolved == null) return null;
    const { xChannel, yChannel } = line;
    const hasX = primitive.isNonZero(xChannel);
    if (resolved.variant === "dynamic") {
      const keepFor = Number(resolved.span.valueOf()) * 3;
      const { span: timeSpan } = resolved;
      return {
        x: telem.streamChannelData({
          timeSpan,
          channel: hasX ? xChannel : yChannel,
          useIndexOfChannel: !hasX,
          keepFor,
        }),
        y: telem.streamChannelData({
          timeSpan,
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
      key={key}
      aetherKey={key}
      x={telemetry.x}
      y={telemetry.y}
      visible={visible}
      legendGroup={line.yAxis.toUpperCase()}
      {...line}
    />
  );
};

interface TitleProps {
  pKey: lineplot.Key;
  editable: boolean;
}

const Title = ({ pKey: key, editable }: TitleProps): ReactElement | null => {
  const { visible, level } = useSelectTitle({ key });
  const name = useSelectName({ key });
  const { update: rn } = useRename({});
  const handleChange = useCallback((name: string) => rn({ key, name }), [rn, key]);
  if (!visible) return null;
  return (
    <BaseTitle
      value={name}
      onChange={handleChange}
      disabled={!editable}
      level={level}
    />
  );
};

interface LegendProps {
  pKey: lineplot.Key;
  variant?: BaseLegendProps["variant"];
  editable: boolean;
  onLineVisibleChange?: (lineKey: string, visible: boolean) => void;
}

const Legend = ({
  pKey: key,
  variant,
  editable,
  onLineVisibleChange,
}: LegendProps): ReactElement | null => {
  const { dispatch } = useDispatch();
  const legend = useSelectLegend({ key });
  const handlePositionChange = useCallback(
    (position: typeof legend.position) =>
      editable &&
      dispatch({ key, actions: [lineplot.setLegendPosition({ position })] }),
    [dispatch, key, editable],
  );
  const handleLineColorChange = useCallback(
    (lineKey: string, color: color.Color) =>
      editable &&
      dispatch({ key, actions: [lineplot.setLineColor({ key: lineKey, color })] }),
    [dispatch, key, editable],
  );
  const handleLineLabelChange = useCallback(
    (lineKey: string, label: string) =>
      editable &&
      dispatch({ key, actions: [lineplot.setLineLabel({ key: lineKey, label })] }),
    [dispatch, key, editable],
  );
  if (legend.hidden) return null;
  return (
    <BaseLegend
      onLineColorChange={handleLineColorChange}
      onLineLabelChange={handleLineLabelChange}
      onLineVisibleChange={onLineVisibleChange}
      position={legend.position}
      onPositionChange={handlePositionChange}
      variant={variant}
    />
  );
};

interface AxisChildrenProps {
  pKey: lineplot.Key;
  editable: boolean;
  resolvedRanges?: Map<string, ResolvedRange>;
  hiddenLines?: Set<string>;
  onSelectRule?: (key: string) => void;
}

interface RuleProps {
  pKey: lineplot.Key;
  ruleKey: string;
  onSelectRule?: (key: string) => void;
}

const Rule = ({ pKey, ruleKey, onSelectRule }: RuleProps): ReactElement | null => {
  const { dispatch } = useDispatch();
  const { key, ...rule } = useSelectRule({ key: pKey, ruleKey });
  const apply = useCallback(
    (action: lineplot.Action): void => {
      dispatch({ key: pKey, actions: [action] });
    },
    [dispatch, pKey],
  );
  const handleLabelChange = useCallback(
    (label: string) => apply(lineplot.setRuleLabel({ key: ruleKey, label })),
    [apply, ruleKey],
  );
  const handlePositionChange = useCallback(
    (position: number) => apply(lineplot.setRulePosition({ key: ruleKey, position })),
    [apply, ruleKey],
  );
  const handleUnitsChange = useCallback(
    (units: string) => apply(lineplot.setRuleUnits({ key: ruleKey, units })),
    [apply, ruleKey],
  );
  const handleClick = useCallback(
    () => onSelectRule?.(ruleKey),
    [onSelectRule, ruleKey],
  );
  return (
    <BaseRule.Rule
      key={key}
      aetherKey={key}
      {...rule}
      onLabelChange={handleLabelChange}
      onPositionChange={handlePositionChange}
      onUnitsChange={handleUnitsChange}
      onClick={handleClick}
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
  pKey: key,
  axisKey,
  editable,
  resolvedRanges,
  hiddenLines,
  onSelectRule,
}: YAxisProps): ReactElement => {
  const { dispatch } = useDispatch();
  const { axis, lineKeys, channels } = useSelectYAxis({ key, axisKey });
  const handleDrop = useCallback(
    (axisKey: lineplot.YAxisKey, dropped: channel.Key[]): void => {
      if (!editable) return;
      const existing = new Set(channels);
      const additions = dropped.filter((channel) => !existing.has(channel));
      if (additions.length > 0)
        dispatch({
          key,
          actions: additions.map((channel) =>
            lineplot.addChannel({ axisKey, channel }),
          ),
        });
    },
    [channels, dispatch, key, editable],
  );
  const handleLabelChange = useCallback(
    (label: string) =>
      editable &&
      dispatch({
        key,
        actions: [lineplot.setAxisLabel({ key: axisKey, label })],
      }),
    [editable],
  );
  const dropProps = useAxisDrop(axisKey, "y", handleDrop);
  const dragging = Haul.useDraggingState();
  const { key: _axisKey, ...axisConfig } = axis;
  return (
    <BaseYAxis
      {...axisConfig}
      {...dropProps}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      showGrid={axisKey === "y1"}
      className={CSS(CSS.dropRegion(canDropHaulItem(dragging)))}
      onLabelChange={handleLabelChange}
    >
      {lineKeys.map((lineKey) => (
        <Line
          key={lineKey}
          pKey={key}
          lineKey={lineKey}
          resolved={resolvedRanges?.get(lineplot.parseLineKey(lineKey).range)}
          visible={hiddenLines == null || !hiddenLines.has(lineKey)}
        />
      ))}
      <Rules pKey={key} axisKey={axisKey} onSelectRule={onSelectRule} />
    </BaseYAxis>
  );
};

interface XAxisProps extends AxisChildrenProps {
  axisKey: lineplot.XAxisKey;
  rangeProviderProps?: Range.ProviderProps;
}

const XAxis = ({
  pKey: key,
  axisKey,
  editable,
  resolvedRanges,
  hiddenLines,
  onSelectRule,
  rangeProviderProps,
}: XAxisProps): ReactElement => {
  const { dispatch } = useDispatch();
  const handleDrop = useCallback(
    (axisKey: lineplot.XAxisKey, [channel]: channel.Key[]): void => {
      if (editable && channel != null)
        dispatch({
          key,
          actions: [lineplot.setXChannel({ axisKey, channel })],
        });
    },
    [dispatch, key, editable],
  );
  const dropProps = useAxisDrop(axisKey, "x", handleDrop);
  const dragging = Haul.useDraggingState();
  const { key: _, ...axisConfig } = useSelectXAxis({ key, axisKey });
  const yAxes = useSelectYAxisKeys({ key });
  const handleLabelChange = useCallback(
    (label: string) =>
      dispatch({
        key,
        actions: [lineplot.setAxisLabel({ key: axisKey, label })],
      }),
    [dispatch, key, axisKey],
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
          pKey={key}
          axisKey={yAxisKey}
          editable={editable}
          resolvedRanges={resolvedRanges}
          hiddenLines={hiddenLines}
          onSelectRule={onSelectRule}
        />
      ))}
      <Rules pKey={key} axisKey={axisKey} onSelectRule={onSelectRule} />
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
  }, [lineCount, hold, prevLineCount, prevHold]);
  return viewportRef;
};

export interface LinePlotProps extends FrameProps {
  editable?: boolean;
  enableTriggers?: boolean | (() => boolean);
  resolvedRanges?: Map<string, ResolvedRange>;
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
  hiddenLines?: Set<string>;
  onLineVisibleChange?: (lineKey: string, visible: boolean) => void;
}

export const LinePlot = ({
  editable = true,
  enableTriggers = true,
  resolvedRanges,
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
  hiddenLines,
  onLineVisibleChange,
  children,
  ref,
  ...rest
}: LinePlotProps): ReactElement => {
  const key = useKey();
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
          resolvedRanges={resolvedRanges}
          hiddenLines={hiddenLines}
          onSelectRule={onSelectRule}
          rangeProviderProps={rangeProviderProps}
        />
      ))}
      <Legend
        pKey={key}
        variant={legendVariant}
        editable={editable}
        onLineVisibleChange={onLineVisibleChange}
      />
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

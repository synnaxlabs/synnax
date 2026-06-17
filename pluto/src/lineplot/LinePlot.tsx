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
  useEnsureRetrieved,
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
  useSingleDispatch,
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

const UNDO_REDO_CONFIG: Triggers.ModeConfig<"undo" | "redo" | "default"> = {
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  default: [],
  defaultMode: "default",
};
const UNDO_REDO_TRIGGERS = Triggers.flattenConfig(UNDO_REDO_CONFIG);

interface UseUndoRedoTriggersProps {
  enabled: boolean | (() => boolean);
}

const useUndoRedoTriggers = ({ enabled }: UseUndoRedoTriggersProps) => {
  const key = useKey();
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
  lineKey: string;
  resolved?: ResolvedRange;
  visible?: boolean;
}

const Line = ({
  lineKey,
  resolved,
  visible = true,
}: LineProps): ReactElement | null => {
  const { key, ...line } = useSelectLine({ lineKey });
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
  editable: boolean;
}

const Title = ({ editable }: TitleProps): ReactElement | null => {
  const { visible, level } = useSelectTitle({});
  const name = useSelectName({});
  const dispatch = useSingleDispatch();
  const handleChange = useCallback(
    (name: string) => dispatch(lineplot.rename({ name })),
    [dispatch],
  );
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
  variant?: BaseLegendProps["variant"];
  editable: boolean;
  onLineVisibleChange?: (lineKey: string, visible: boolean) => void;
}

const Legend = ({
  variant,
  editable,
  onLineVisibleChange,
}: LegendProps): ReactElement | null => {
  const dispatch = useSingleDispatch();
  const legend = useSelectLegend({});
  const handlePositionChange = useCallback(
    (position: typeof legend.position) =>
      editable && dispatch(lineplot.setLegendPosition({ position })),
    [dispatch, editable],
  );
  const handleLineColorChange = useCallback(
    (lineKey: string, color: color.Color) =>
      editable && dispatch(lineplot.setLineColor({ key: lineKey, color })),
    [dispatch, editable],
  );
  const handleLineLabelChange = useCallback(
    (lineKey: string, label: string) =>
      editable && dispatch(lineplot.setLineLabel({ key: lineKey, label })),
    [dispatch, editable],
  );
  if (!legend.visible) return null;
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
  editable: boolean;
  resolvedRanges?: Map<string, ResolvedRange>;
  hiddenLines?: Set<string>;
  onSelectRule?: (key: string) => void;
}

interface RuleProps {
  ruleKey: string;
  onSelectRule?: (key: string) => void;
}

const Rule = ({ ruleKey: key, onSelectRule }: RuleProps): ReactElement | null => {
  const dispatch = useSingleDispatch();
  const { key: _, ...rule } = useSelectRule({ ruleKey: key });
  const handleLabelChange = useCallback(
    (label: string) => dispatch(lineplot.setRuleLabel({ key, label })),
    [key],
  );
  const handlePositionChange = useCallback(
    (position: number) => dispatch(lineplot.setRulePosition({ key, position })),
    [key],
  );
  const handleUnitsChange = useCallback(
    (units: string) => dispatch(lineplot.setRuleUnits({ key, units })),
    [key],
  );
  const handleClick = useCallback(() => onSelectRule?.(key), [onSelectRule, key]);
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
  axisKey: lineplot.AxisKey;
  onSelectRule?: (key: string) => void;
}

const Rules = ({ axisKey, onSelectRule }: RulesProps): ReactElement => {
  const ruleKeys = useSelectAxisRuleKeys({ axisKey });
  return (
    <>
      {ruleKeys.map((ruleKey) => (
        <Rule key={ruleKey} ruleKey={ruleKey} onSelectRule={onSelectRule} />
      ))}
    </>
  );
};

interface YAxisProps extends AxisChildrenProps {
  axisKey: lineplot.YAxisKey;
}

const YAxis = ({
  axisKey,
  editable,
  resolvedRanges,
  hiddenLines,
  onSelectRule,
}: YAxisProps): ReactElement => {
  const dispatch = useSingleDispatch();
  const { axis, lineKeys, channels } = useSelectYAxis({ axisKey });
  const handleDrop = useCallback(
    (axisKey: lineplot.YAxisKey, dropped: channel.Key[]): void => {
      if (!editable) return;
      const existing = new Set(channels);
      const additions = dropped.filter((channel) => !existing.has(channel));
      if (additions.length > 0)
        dispatch(additions.map((channel) => lineplot.addChannel({ axisKey, channel })));
    },
    [channels, dispatch, editable],
  );
  const handleLabelChange = useCallback(
    (label: string) =>
      editable && dispatch(lineplot.setAxisLabel({ key: axisKey, label })),
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
          lineKey={lineKey}
          resolved={resolvedRanges?.get(lineplot.parseLineKey(lineKey).range)}
          visible={hiddenLines == null || !hiddenLines.has(lineKey)}
        />
      ))}
      <Rules axisKey={axisKey} onSelectRule={onSelectRule} />
    </BaseYAxis>
  );
};

interface XAxisProps extends AxisChildrenProps {
  axisKey: lineplot.XAxisKey;
  rangeProviderProps?: Range.ProviderProps;
}

const XAxis = ({
  axisKey,
  editable,
  resolvedRanges,
  hiddenLines,
  onSelectRule,
  rangeProviderProps,
}: XAxisProps): ReactElement => {
  const dispatch = useSingleDispatch();
  const handleDrop = useCallback(
    (axisKey: lineplot.XAxisKey, [channel]: channel.Key[]): void => {
      if (editable && channel != null)
        dispatch(lineplot.setXChannel({ axisKey, channel }));
    },
    [dispatch, editable],
  );
  const dropProps = useAxisDrop(axisKey, "x", handleDrop);
  const dragging = Haul.useDraggingState();
  const { key: _, ...axisConfig } = useSelectXAxis({ axisKey });
  const yAxes = useSelectYAxisKeys({});
  const handleLabelChange = useCallback(
    (label: string) => dispatch(lineplot.setAxisLabel({ key: axisKey, label })),
    [dispatch, axisKey],
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
          axisKey={yAxisKey}
          editable={editable}
          resolvedRanges={resolvedRanges}
          hiddenLines={hiddenLines}
          onSelectRule={onSelectRule}
        />
      ))}
      <Rules axisKey={axisKey} onSelectRule={onSelectRule} />
      <Range.Provider {...rangeProviderProps} />
    </BaseXAxis>
  );
};

interface UseViewportResetParams {
  hold?: boolean;
}

const useViewportReset = ({
  hold,
}: UseViewportResetParams): RefObject<Viewport.UseRefValue | null> => {
  const lineCount = useSelectLineCount({});
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
  useUndoRedoTriggers({ enabled: enableTriggers });
  const xAxisKeys = useSelectXAxisKeys({});
  const viewportRef = useViewportReset({ hold: rest.hold });
  return (
    <Frame ref={ref} {...rest}>
      {xAxisKeys.map((xAxisKey) => (
        <XAxis
          key={xAxisKey}
          axisKey={xAxisKey}
          editable={editable}
          resolvedRanges={resolvedRanges}
          hiddenLines={hiddenLines}
          onSelectRule={onSelectRule}
          rangeProviderProps={rangeProviderProps}
        />
      ))}
      <Legend
        variant={legendVariant}
        editable={editable}
        onLineVisibleChange={onLineVisibleChange}
      />
      <Title editable={editable} />
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

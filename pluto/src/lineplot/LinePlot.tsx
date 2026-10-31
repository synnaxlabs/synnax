// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import {
  box,
  type color,
  type location,
  primitive,
  resolveWindow,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import { type ReactElement, type RefObject, useEffect, useRef } from "react";

import { Channel } from "@/channel";
import { CSS } from "@/css";
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
  type DerivedLine,
  useAxisRuleKeys,
  useDispatch,
  useLegend,
  useLine,
  useLineCount,
  useName,
  useRedo,
  useRename,
  useRule,
  useTitle,
  useUndo,
  useXAxis,
  useXAxisKeys,
  useYAxis,
  useYAxisKeys,
} from "@/lineplot/queries";
import { Range } from "@/lineplot/range";
import { Rule as BaseRule } from "@/lineplot/rule";
import { useKey } from "@/lineplot/Suspended";
import { Title as BaseTitle } from "@/lineplot/Title";
import { Tooltip } from "@/lineplot/tooltip";
import { Viewport as BaseViewport } from "@/lineplot/Viewport";
import { Ranger } from "@/ranger";
import { telem } from "@/telem/aether";
import { type Viewport } from "@/viewport";

export const axisLabel = (key: lineplot.AxisKey): string => key.toUpperCase();

const AXIS_LOCATIONS: Record<lineplot.AxisKey, location.Outer> = {
  y1: "left",
  y2: "right",
  y3: "left",
  y4: "right",
  x1: "bottom",
  x2: "top",
};

interface UseUndoRedoTriggersProps {
  key: lineplot.Key;
  enabled: Triggers.Condition;
}

const useUndoRedoTriggers = ({ key, enabled }: UseUndoRedoTriggersProps) => {
  const { undo } = useUndo({ key });
  const { redo } = useRedo({ key });
  Triggers.useUndoRedo({ undo, redo, enabled });
};

/** @returns a function that applies actions as one undo step on an editable plot. */
const useEdit = (
  key: lineplot.Key,
  editable: boolean,
): ((actions: lineplot.Action[]) => void) => {
  const { dispatch } = useDispatch();
  return (actions: lineplot.Action[]): void => {
    if (editable && actions.length > 0) dispatch({ key, actions });
  };
};

const addChannels = (
  axisKey: lineplot.YAxisKey,
  items: Channel.HaulItem[],
): lineplot.Action[] =>
  items.map(({ key }) => lineplot.addChannel({ axisKey, channel: key }));

const addRanges = (items: Ranger.HaulItem[]): lineplot.Action[] =>
  items.map(({ key }) => lineplot.addRange({ range: { variant: "key", key } }));

const canDropChannelOrRange: Haul.CanDrop = (state) =>
  Channel.canDropHaulItem(state) || Ranger.canDropHaulItem(state);

interface LineProps {
  pKey: lineplot.Key;
  lineKey: string;
  windows?: Window[];
  visible?: boolean;
}

const GROUPS_PER_COLUMN: Record<lineplot.Detail, number> = {
  low: 1 / 4,
  medium: 1 / 2,
  high: 1,
};

// A live window of a minute or less draws from the stream at once, so its lines report
// no loading while they have no data.
const LOADING_MIN_SPAN = Number(TimeSpan.minutes(1).valueOf());

const createTelemetry = (
  window: Window,
  xChannel: channel.Key,
  yChannel: channel.Key,
  aggregation: lineplot.Aggregation,
) => {
  const hasX = primitive.isNonZero(xChannel);
  const props = {
    window,
    xChannel,
    aggregation,
  };
  return {
    x: telem.channelData({
      ...props,
      channel: hasX ? xChannel : yChannel,
      useIndexOfChannel: !hasX,
    }),
    y: telem.channelData({ ...props, channel: yChannel }),
  };
};

// Line draws a stored line as one trace per window the plot shows, styled alike. Only
// the first trace has a legend entry, under the stored line's key.
const Line = ({
  pKey,
  lineKey,
  windows = [],
  visible = true,
}: LineProps): ReactElement => {
  const line = useLine({ key: pKey, lineKey });
  return (
    <>
      {windows.map((window, i) => (
        <Trace key={i} line={line} window={window} index={i} visible={visible} />
      ))}
    </>
  );
};

interface TraceProps {
  line: DerivedLine;
  window: Window;
  index: number;
  visible: boolean;
}

const Trace = ({ line, window, index, visible }: TraceProps): ReactElement => {
  const { key, aggregation, detail, ...rest } = line;
  // JSX takes the rest object, so the compiler counts reads from it as changing later.
  const { xChannel, yChannel } = line;
  const telemetry = createTelemetry(window, xChannel, yChannel, aggregation);
  const aetherKey = index === 0 ? key : `${key}/${index}`;
  return (
    <BaseLine
      key={aetherKey}
      aetherKey={aetherKey}
      x={telemetry.x}
      y={telemetry.y}
      visible={visible}
      legend={index === 0}
      groupsPerColumn={GROUPS_PER_COLUMN[detail]}
      loadingHidden={
        window.variant === "dynamic" && window.timeSpan <= LOADING_MIN_SPAN
      }
      legendGroup={line.yAxis.toUpperCase()}
      {...rest}
    />
  );
};

interface TitleProps {
  pKey: lineplot.Key;
  editable: boolean;
}

const Title = ({ pKey: key, editable }: TitleProps): ReactElement | null => {
  const { visible, level } = useTitle({ key });
  const name = useName({ key });
  const { update: rn } = useRename({});
  const handleChange = (name: string) => rn({ key, name });
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
  const legend = useLegend({ key });
  const handlePositionChange = (position: typeof legend.position) =>
    editable && dispatch({ key, actions: [lineplot.setLegendPosition({ position })] });
  const handleLineColorChange = (lineKey: string, color: color.Color) =>
    editable &&
    dispatch({ key, actions: [lineplot.setLineColor({ key: lineKey, color })] });
  const handleLineLabelChange = (lineKey: string, label: string) =>
    editable &&
    dispatch({ key, actions: [lineplot.setLineLabel({ key: lineKey, label })] });
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
  windows?: Window[];
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
  const { key, ...rule } = useRule({ key: pKey, ruleKey });
  const apply = (action: lineplot.Action): void => {
    dispatch({ key: pKey, actions: [action] });
  };
  const handleLabelChange = (label: string) =>
    apply(lineplot.setRuleLabel({ key: ruleKey, label }));
  const handlePositionChange = (position: number) =>
    apply(lineplot.setRulePosition({ key: ruleKey, position }));
  const handleUnitsChange = (units: string) =>
    apply(lineplot.setRuleUnits({ key: ruleKey, units }));
  const handleClick = () => onSelectRule?.(ruleKey);
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
  const ruleKeys = useAxisRuleKeys({ key: pKey, axisKey });
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
  windows,
  hiddenLines,
  onSelectRule,
}: YAxisProps): ReactElement => {
  const { dispatch } = useDispatch();
  const { axis, lineKeys } = useYAxis({ key, axisKey });
  const edit = useEdit(key, editable);
  const dropProps = Haul.useDrop({
    type: "channel_lineplot_y_axis",
    canDrop: Channel.canDropHaulItem,
    onDrop: ({ items }) => {
      const channels = Channel.filterHaulItems(items);
      edit(addChannels(axisKey, channels));
      return channels;
    },
  });
  const handleLabelChange = (label: string) =>
    editable &&
    dispatch({
      key,
      actions: [lineplot.setAxisLabel({ key: axisKey, label })],
    });
  const dragging = Haul.useDraggingState();
  const { key: _axisKey, ...axisConfig } = axis;
  return (
    <BaseYAxis
      {...axisConfig}
      {...dropProps}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      showGrid={axisKey === "y1"}
      className={CSS.cls(CSS.dropRegion(Channel.canDropHaulItem(dragging)))}
      onLabelChange={handleLabelChange}
    >
      {lineKeys.map((lineKey) => (
        <Line
          key={lineKey}
          pKey={key}
          lineKey={lineKey}
          windows={windows}
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
  windows,
  hiddenLines,
  onSelectRule,
  rangeProviderProps,
}: XAxisProps): ReactElement => {
  const { dispatch } = useDispatch();
  const edit = useEdit(key, editable);
  const dropProps = Haul.useDrop({
    type: "lineplot_x_axis",
    canDrop: canDropChannelOrRange,
    onDrop: ({ items }) => {
      const channels = Channel.filterHaulItems(items);
      const ranges = Ranger.filterHaulItems(items);
      // An X axis has one channel, so only the first dropped channel applies.
      const setChannel = channels
        .slice(0, 1)
        .map(({ key }) => lineplot.setXChannel({ axisKey, channel: key }));
      edit([...setChannel, ...addRanges(ranges)]);
      return [...channels, ...ranges];
    },
  });
  const dragging = Haul.useDraggingState();
  const { key: _, ...axisConfig } = useXAxis({ key, axisKey });
  const yAxes = useYAxisKeys({ key });
  const handleLabelChange = (label: string) =>
    dispatch({
      key,
      actions: [lineplot.setAxisLabel({ key: axisKey, label })],
    });
  return (
    <BaseXAxis
      {...axisConfig}
      {...dropProps}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      className={CSS.cls(CSS.dropRegion(canDropChannelOrRange(dragging)))}
      showGrid={axisKey === "x1"}
      onLabelChange={handleLabelChange}
    >
      {yAxes.map((yAxisKey) => (
        <YAxis
          key={yAxisKey}
          pKey={key}
          axisKey={yAxisKey}
          editable={editable}
          windows={windows}
          hiddenLines={hiddenLines}
          onSelectRule={onSelectRule}
        />
      ))}
      <Rules pKey={key} axisKey={axisKey} onSelectRule={onSelectRule} />
      <Range.Provider {...rangeProviderProps} />
    </BaseXAxis>
  );
};

const FIRST_Y_AXIS: lineplot.YAxisKey = "y1";

/**
 * Adds a channel dropped on the plot area to the first Y axis, and a range to the plot.
 */
const usePlotDrop = (key: lineplot.Key, editable: boolean): Haul.UseDropReturn => {
  const edit = useEdit(key, editable);
  return Haul.useDrop({
    type: "lineplot_plot",
    canDrop: canDropChannelOrRange,
    onDrop: ({ items }) => {
      const channels = Channel.filterHaulItems(items);
      const ranges = Ranger.filterHaulItems(items);
      edit([...addChannels(FIRST_Y_AXIS, channels), ...addRanges(ranges)]);
      return [...channels, ...ranges];
    },
  });
};

interface UseViewportResetParams {
  key: lineplot.Key;
  hold?: boolean;
}

const useViewportReset = ({
  key,
  hold,
}: UseViewportResetParams): RefObject<Viewport.UseRefValue | null> => {
  const lineCount = useLineCount({ key });
  const viewportRef = useRef<Viewport.UseRefValue | null>(null);
  const prevLineCount = useRef(lineCount);
  const prevHold = useRef(hold);
  useEffect(() => {
    const linesAppeared = prevLineCount.current === 0 && lineCount !== 0;
    const holdReleased = prevHold.current === true && hold === false;
    prevLineCount.current = lineCount;
    prevHold.current = hold;
    if (linesAppeared || holdReleased) viewportRef.current?.reset();
  }, [lineCount, hold]);
  return viewportRef;
};

export interface LinePlotProps extends FrameProps {
  editable?: boolean;
  enableTriggers?: Triggers.Condition;
  /** The windows of the plot's ranges, in order. */
  windows?: Window[];
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

const fetchingMessage = (windows: Window[] | undefined): string | undefined => {
  if (windows == null || windows.length === 0) return undefined;
  const now = TimeStamp.now();
  const spans = windows.map((w) => resolveWindow(w, now).span);
  const longest = spans.reduce((a, b) => (b.greaterThan(a) ? b : a));
  return `Fetching ${longest.toString()} of data`;
};

export const LinePlot = ({
  editable = true,
  enableTriggers = true,
  windows,
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
  const xAxisKeys = useXAxisKeys({ key });
  const viewportRef = useViewportReset({ key, hold: rest.hold });
  const plotDropProps = usePlotDrop(key, editable);
  return (
    <Frame ref={ref} loadingMessage={fetchingMessage(windows)} {...rest}>
      {xAxisKeys.map((xAxisKey) => (
        <XAxis
          key={xAxisKey}
          pKey={key}
          axisKey={xAxisKey}
          editable={editable}
          windows={windows}
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
        {...plotDropProps}
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

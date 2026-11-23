// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot, type ranger } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import {
  box,
  color,
  type location,
  primitive,
  resolveWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import {
  createContext,
  type ReactElement,
  type RefObject,
  use,
  useEffect,
  useRef,
  useState,
} from "react";

import { Channel } from "@/channel";
import { XAxis as BaseXAxis, YAxis as BaseYAxis } from "@/lineplot/Axis";
import { Frame, type FrameProps } from "@/lineplot/Frame";
import {
  Legend as BaseLegend,
  type LegendProps as BaseLegendProps,
} from "@/lineplot/Legend";
import { Line as BaseLine } from "@/lineplot/Line";
import { Measure } from "@/lineplot/measure";
import type { measure } from "@/lineplot/measure/aether";
import {
  type DerivedLine,
  useAxisRuleKeys,
  useDispatch,
  useLegend,
  useLine,
  useLineCount,
  useName,
  useRanges,
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
import type { tooltip } from "@/lineplot/tooltip/aether";
import { Viewport as BaseViewport } from "@/lineplot/Viewport";
import { Ranger } from "@/ranger";
import { telem } from "@/telem/aether";
import type { Viewport } from "@/viewport";

export const axisLabel = (key: lineplot.AxisKey): string => key.toUpperCase();

const AXIS_LOCATIONS: Record<lineplot.AxisKey, location.Outer> = {
  y1: "left",
  y2: "right",
  y3: "left",
  y4: "right",
  x1: "bottom",
  x2: "top",
};

const LOCATION_ICONS: Record<location.Outer, Icon.ReactElement> = {
  left: <Icon.Axis.Left />,
  right: <Icon.Axis.Right />,
  top: <Icon.Axis.Top />,
  bottom: <Icon.Axis.Bottom />,
};

/** @returns an icon showing the side of the plot an axis sits on. */
export const axisIcon = (key: lineplot.AxisKey): Icon.ReactElement =>
  LOCATION_ICONS[AXIS_LOCATIONS[key]];

interface UseUndoRedoTriggersProps {
  key: lineplot.Key;
  enabled: Triggers.Condition;
}

const useUndoRedoTriggers = ({ key, enabled }: UseUndoRedoTriggersProps) => {
  const { undo } = useUndo({ key });
  const { redo } = useRedo({ key });
  Triggers.useUndoRedo({ undo, redo, enabled });
};

/**
 * @returns a function that applies actions as one undo step on an editable plot, and
 * returns false when the plot is not editable.
 */
const useEdit = (
  key: lineplot.Key,
  editable: boolean,
): ((actions: lineplot.Action[]) => boolean) => {
  const { dispatch } = useDispatch();
  return (actions: lineplot.Action[]): boolean => {
    if (!editable) return false;
    if (actions.length > 0) dispatch({ key, actions });
    return true;
  };
};

// A drag source removes the items that a drop reports, so a drop reports only the
// channels and ranges that the plot did not have.
const newChannels = (
  items: Channel.HaulItem[],
  existing: channel.Key[],
): Channel.HaulItem[] =>
  items.filter(
    ({ key }, i) =>
      !existing.includes(key) && items.findIndex((item) => item.key === key) === i,
  );

const newRanges = (
  items: Ranger.HaulItem[],
  existing: ranger.Ref[],
): Ranger.HaulItem[] =>
  items.filter(
    ({ key }, i) =>
      !existing.some((r) => r.variant === "key" && r.key === key) &&
      items.findIndex((item) => item.key === key) === i,
  );

const addChannels = (
  axisKey: lineplot.YAxisKey,
  items: Channel.HaulItem[],
): lineplot.Action[] =>
  items.map(({ key }) => lineplot.addChannel({ axisKey, channel: key }));

const addRanges = (items: Ranger.HaulItem[]): lineplot.Action[] =>
  items.map(({ key }) => lineplot.addRange({ range: { variant: "key", key } }));

interface DropHighlight extends Omit<Haul.UseDragOverReturn, "over"> {
  /** Set while a drag that the element can take is in progress. */
  "data-droppable"?: true;
  /** Set while a drag that the element can take is over it. */
  "data-drag-over"?: true;
}

/**
 * @returns props that mark a drop target while it can take the dragged items, and more
 * strongly while the drag is over it.
 */
const useDropHighlight = (canDrop: Haul.CanDrop): DropHighlight => {
  const { over, ...dragOverProps } = Haul.useDragOver();
  const droppable = canDrop(Haul.useDraggingState());
  return {
    ...dragOverProps,
    "data-droppable": droppable ? true : undefined,
    "data-drag-over": droppable && over ? true : undefined,
  };
};

const canDropChannelOrRange: Haul.CanDrop = (state) =>
  Channel.canDropHaulItem(state) || Ranger.canDropHaulItem(state);

/** @returns whether an x axis draws each range from its zero. */
export const isRelative = ({ type, timeMode }: lineplot.Axis): boolean =>
  type === "time" && timeMode === "relative";

/** A range that a line plot shows. */
export interface RangeSpec {
  /** Identifies the range. Unique among the ranges of a plot. */
  key: string;
  window: Window;
  /** The name of the range on a relative x axis. Defaults to key. */
  label?: string;
  /** The color of each line of the range on a relative x axis. */
  color?: color.Crude;
  /**
   * The time that a relative x axis draws at zero. A range with no zero does not draw
   * on a relative axis.
   */
  zero?: TimeStamp;
  /** True when zero is not the start of the range. */
  customZero?: boolean;
}

/** The line and range of a line on a relative x axis. */
export interface TraceKey {
  line: string;
  range: string;
}

/** @returns the key of the line that draws a stored line for a range. */
export const traceKey = ({ line, range }: TraceKey): string => `${line}/${range}`;

// Hover is the legend entry under the pointer: a line, a range, or a line of a range.
interface Hover {
  range?: string;
  line?: string;
}

interface HoverContextValue {
  hover: Hover | null;
  hiddenRanges?: Set<string>;
  hiddenTraces?: Set<string>;
}

const HoverContext = createContext<HoverContextValue>({ hover: null });
HoverContext.displayName = "LinePlot.HoverContext";

// The opacity of the lines that the hovered legend entry does not name.
const DIMMED_OPACITY = 0.25;

interface LineProps {
  pKey: lineplot.Key;
  lineKey: string;
  ranges?: RangeSpec[];
  relative: boolean;
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

// Line draws a stored line as one trace per range the plot shows. On an absolute x axis
// the traces are styled alike and split the time between them, and only the first has
// a legend entry, under the stored line's key. On a relative axis each trace takes its
// range's color and has its own legend entry under the range.
const Line = ({
  pKey,
  lineKey,
  ranges = [],
  relative,
  visible = true,
}: LineProps): ReactElement => {
  const line = useLine({ key: pKey, lineKey });
  const { hover, hiddenRanges, hiddenTraces } = use(HoverContext);
  if (!relative) {
    const hovered = hover?.line === line.key;
    return (
      <>
        {ranges.map((range, i) => (
          <Trace
            key={i}
            line={line}
            window={range.window}
            aetherKey={i === 0 ? line.key : `${line.key}/${i}`}
            group={line.key}
            legend={i === 0}
            legendGroup={line.yAxis.toUpperCase()}
            visible={visible}
            opacity={hover != null && !hovered ? DIMMED_OPACITY : 1}
            raised={hovered}
          />
        ))}
      </>
    );
  }
  return (
    <>
      {ranges.map((range) => {
        if (range.zero == null) return null;
        const key = traceKey({ line: line.key, range: range.key });
        const shown =
          visible &&
          hiddenRanges?.has(range.key) !== true &&
          hiddenTraces?.has(key) !== true;
        const hovered =
          hover != null &&
          hover.range === range.key &&
          (hover.line == null || hover.line === line.key);
        const c = color.construct(range.color ?? line.color);
        return (
          <Trace
            key={key}
            line={line}
            window={range.window}
            aetherKey={key}
            legend
            legendGroup={range.key}
            visible={shown}
            zero={range.zero}
            tooltipGroup={line.key}
            tooltipLabel={range.label ?? range.key}
            color={c}
            opacity={hover != null && hover.range !== range.key ? DIMMED_OPACITY : 1}
            raised={hovered}
          />
        );
      })}
    </>
  );
};

interface TraceProps {
  line: DerivedLine;
  window: Window;
  aetherKey: string;
  legend: boolean;
  legendGroup: string;
  visible: boolean;
  zero?: TimeStamp;
  group?: string;
  tooltipGroup?: string;
  tooltipLabel?: string;
  color?: color.Color;
  opacity?: number;
  raised?: boolean;
}

const Trace = ({
  line,
  window,
  aetherKey,
  legend,
  legendGroup,
  visible,
  zero,
  group,
  tooltipGroup,
  tooltipLabel,
  color: traceColor,
  opacity,
  raised = false,
}: TraceProps): ReactElement => {
  const { key, aggregation, detail, ...rest } = line;
  // JSX takes the rest object, so the compiler counts reads from it as changing later.
  const { xChannel, yChannel, strokeWidth } = line;
  const telemetry = createTelemetry(window, xChannel, yChannel, aggregation);
  return (
    <BaseLine
      key={aetherKey}
      aetherKey={aetherKey}
      x={telemetry.x}
      y={telemetry.y}
      visible={visible}
      legend={legend}
      groupsPerColumn={GROUPS_PER_COLUMN[detail]}
      loadingHidden={
        window.variant === "dynamic" && window.timeSpan <= LOADING_MIN_SPAN
      }
      legendGroup={legendGroup}
      {...rest}
      axis={line.yAxis}
      color={traceColor ?? line.color}
      strokeWidth={raised ? strokeWidth * 2 : strokeWidth}
      zero={zero}
      group={group}
      tooltipGroup={tooltipGroup}
      tooltipLabel={tooltipLabel}
      opacity={opacity}
      raised={raised}
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

interface LegendProps extends Pick<
  BaseLegendProps,
  | "ranges"
  | "hiddenRanges"
  | "onRangeColorChange"
  | "onRangeVisibleChange"
  | "onHoverChange"
> {
  pKey: lineplot.Key;
  editable: boolean;
  onLineVisibleChange?: BaseLegendProps["onLineVisibleChange"];
}

const Legend = ({
  pKey: key,
  editable,
  onLineVisibleChange,
  ranges,
  ...rest
}: LegendProps): ReactElement | null => {
  const { dispatch } = useDispatch();
  const legend = useLegend({ key });
  const handlePositionChange = (position: typeof legend.position) =>
    editable && dispatch({ key, actions: [lineplot.setLegendPosition({ position })] });
  const handleLineColorChange = (lineKey: string, color: color.Color) =>
    editable &&
    dispatch({
      key,
      actions: [
        lineplot.setLineColor({ key: lineKey, color: { type: "fixed", value: color } }),
      ],
    });
  // On a relative axis each entry is a line of one range, and the label is the line's.
  const handleLineLabelChange = (entryKey: string, label: string) => {
    const line = ranges == null ? entryKey : parseTraceKey(entryKey, ranges)?.line;
    if (!editable || line == null) return;
    dispatch({ key, actions: [lineplot.setLineLabel({ key: line, label })] });
  };
  if (legend.hidden) return null;
  return (
    <BaseLegend
      onLineColorChange={handleLineColorChange}
      onLineLabelChange={handleLineLabelChange}
      onLineVisibleChange={onLineVisibleChange}
      position={legend.position}
      onPositionChange={handlePositionChange}
      ranges={ranges}
      {...rest}
    />
  );
};

interface AxisChildrenProps {
  pKey: lineplot.Key;
  editable: boolean;
  ranges?: RangeSpec[];
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
  relative: boolean;
}

const YAxis = ({
  pKey: key,
  axisKey,
  editable,
  ranges,
  relative,
  hiddenLines,
  onSelectRule,
}: YAxisProps): ReactElement => {
  const { dispatch } = useDispatch();
  const { axis, lineKeys, channels: existing } = useYAxis({ key, axisKey });
  const edit = useEdit(key, editable);
  const highlight = useDropHighlight(Channel.canDropHaulItem);
  const dropProps = Haul.useDrop({
    type: "channel_lineplot_y_axis",
    canDrop: Channel.canDropHaulItem,
    onDrop: ({ items }) => {
      const channels = newChannels(Channel.filterHaulItems(items), existing);
      return edit(addChannels(axisKey, channels)) ? channels : [];
    },
  });
  const handleLabelChange = (label: string) =>
    editable &&
    dispatch({
      key,
      actions: [lineplot.setAxisLabel({ key: axisKey, label })],
    });
  const { key: _axisKey, ...axisConfig } = axis;
  return (
    <BaseYAxis
      {...axisConfig}
      {...dropProps}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      showGrid={axisKey === "y1"}
      {...highlight}
      onLabelChange={handleLabelChange}
    >
      {lineKeys.map((lineKey) => (
        <Line
          key={lineKey}
          pKey={key}
          lineKey={lineKey}
          ranges={ranges}
          relative={relative}
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
  relativeDisabled: boolean;
}

const XAxis = ({
  pKey: key,
  axisKey,
  editable,
  ranges,
  hiddenLines,
  onSelectRule,
  rangeProviderProps,
  relativeDisabled,
}: XAxisProps): ReactElement => {
  const { dispatch } = useDispatch();
  const edit = useEdit(key, editable);
  const highlight = useDropHighlight(canDropChannelOrRange);
  const plotRanges = useRanges({ key });
  const dropProps = Haul.useDrop({
    type: "lineplot_x_axis",
    canDrop: canDropChannelOrRange,
    onDrop: ({ items }) => {
      // An X axis has one channel, so only the first dropped channel applies.
      const channels = Channel.filterHaulItems(items).slice(0, 1);
      const ranges = newRanges(Ranger.filterHaulItems(items), plotRanges);
      const setChannel = channels.map(({ key }) =>
        lineplot.setXChannel({ axisKey, channel: key }),
      );
      return edit([...setChannel, ...addRanges(ranges)])
        ? [...channels, ...ranges]
        : [];
    },
  });
  const axis = useXAxis({ key, axisKey });
  const { key: _, timeMode: __, ...axisConfig } = axis;
  const relative = !relativeDisabled && isRelative(axis);
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
      type={relative ? "relative" : axisConfig.type}
      location={AXIS_LOCATIONS[axisKey]}
      axisKey={axisKey}
      {...highlight}
      showGrid={axisKey === "x1"}
      onLabelChange={handleLabelChange}
    >
      {yAxes.map((yAxisKey) => (
        <YAxis
          key={yAxisKey}
          pKey={key}
          axisKey={yAxisKey}
          editable={editable}
          ranges={ranges}
          relative={relative}
          hiddenLines={hiddenLines}
          onSelectRule={onSelectRule}
        />
      ))}
      <Rules pKey={key} axisKey={axisKey} onSelectRule={onSelectRule} />
      {!relative && (
        <Range.Provider {...rangeProviderProps} breaksDisabled={axisKey !== "x1"} />
      )}
    </BaseXAxis>
  );
};

const FIRST_Y_AXIS: lineplot.YAxisKey = "y1";

interface PlotDrop extends Haul.UseDropReturn, DropHighlight {}

/**
 * Adds a channel dropped on the plot area to the first Y axis, and a range to the plot.
 * Marks the plot area while it can take the dragged items, and more strongly while the
 * drag is over it.
 */
const usePlotDrop = (key: lineplot.Key, editable: boolean): PlotDrop => {
  const edit = useEdit(key, editable);
  const { channels: existing } = useYAxis({ key, axisKey: FIRST_Y_AXIS });
  const plotRanges = useRanges({ key });
  const highlight = useDropHighlight(canDropChannelOrRange);
  const dropProps = Haul.useDrop({
    type: "lineplot_plot",
    canDrop: canDropChannelOrRange,
    onDrop: ({ items }) => {
      const channels = newChannels(Channel.filterHaulItems(items), existing);
      const ranges = newRanges(Ranger.filterHaulItems(items), plotRanges);
      const actions = [...addChannels(FIRST_Y_AXIS, channels), ...addRanges(ranges)];
      return edit(actions) ? [...channels, ...ranges] : [];
    },
  });
  return { ...dropProps, ...highlight };
};

interface UseViewportResetParams {
  key: lineplot.Key;
  relative: boolean;
}

// useViewportReset resets the view when the first lines appear, and when the x axis
// changes between absolute and relative time, because a zoom in one means nothing in
// the other.
const useViewportReset = ({
  key,
  relative,
}: UseViewportResetParams): RefObject<Viewport.UseRefValue | null> => {
  const lineCount = useLineCount({ key });
  const viewportRef = useRef<Viewport.UseRefValue | null>(null);
  const prev = useRef({ lineCount, relative });
  useEffect(() => {
    const linesAppeared = prev.current.lineCount === 0 && lineCount !== 0;
    const modeChanged = prev.current.relative !== relative;
    prev.current = { lineCount, relative };
    if (linesAppeared || modeChanged) viewportRef.current?.reset();
  }, [lineCount, relative]);
  return viewportRef;
};

/** The line and the time under the pointer, on a relative x axis. */
export interface Hovered extends TraceKey {
  /** The time of the nearest sample of the line. */
  time: TimeStamp;
}

export interface LinePlotProps extends FrameProps {
  editable?: boolean;
  enableTriggers?: Triggers.Condition;
  /** The plot's ranges, in order. */
  ranges?: RangeSpec[];
  enableTooltip?: boolean;
  enableMeasure?: boolean;
  measureMode?: measure.Mode;
  onMeasureModeChange?: (mode: measure.Mode) => void;
  initialViewport?: Viewport.UseProps["initial"];
  viewportTriggers?: Viewport.UseProps["triggers"];
  rangeProviderProps?: Range.ProviderProps;
  onSelectRule?: (key: string) => void;
  /**
   * The keys of the stored lines that do not show. On a relative x axis, a hidden line
   * does not show in any range.
   */
  hiddenLines?: Set<string>;
  /** The keys of the ranges that do not show on a relative x axis. */
  hiddenRanges?: Set<string>;
  /** The keys (see traceKey) of the lines of one range that do not show. */
  hiddenTraces?: Set<string>;
  /**
   * Called when the user toggles a legend entry: a stored line on an absolute x axis,
   * or a line of one range (see traceKey) on a relative axis.
   */
  onLineVisibleChange?: (key: string, visible: boolean, isolated: boolean) => void;
  onRangeVisibleChange?: (key: string, visible: boolean, isolated: boolean) => void;
  onRangeColorChange?: (key: string, color: color.Color) => void;
  /** Called with the line under the pointer on a relative x axis, or null. */
  onHoveredChange?: (hovered: Hovered | null) => void;
  /** Draws a relative x axis as absolute. */
  relativeDisabled?: boolean;
}

/** @returns the line and range of a key from traceKey, or null if no range matches. */
export const parseTraceKey = (
  key: string,
  ranges: Pick<RangeSpec, "key">[],
): TraceKey | null => {
  const range = ranges.find((r) => key.endsWith(`/${r.key}`));
  if (range == null) return null;
  return { line: key.slice(0, -range.key.length - 1), range: range.key };
};

interface HoverParams {
  relative: boolean;
  ranges: RangeSpec[];
  hiddenLines?: Set<string>;
  hiddenRanges?: Set<string>;
  hiddenTraces?: Set<string>;
}

// parseHover reads the key of a legend entry: a line on an absolute axis, and a range
// or a line of a range on a relative one. An entry that draws nothing is no hover.
const parseHover = (
  key: string | null,
  { relative, ranges, hiddenLines, hiddenRanges, hiddenTraces }: HoverParams,
): Hover | null => {
  if (key == null) return null;
  if (!relative) return hiddenLines?.has(key) === true ? null : { line: key };
  const hover: Hover | null = ranges.some((r) => r.key === key)
    ? { range: key }
    : parseTraceKey(key, ranges);
  const range = ranges.find((r) => r.key === hover?.range);
  if (hover == null || range?.zero == null || hiddenRanges?.has(range.key) === true)
    return null;
  if (hover.line == null) return hover;
  if (hiddenLines?.has(hover.line) === true || hiddenTraces?.has(key) === true)
    return null;
  return hover;
};

const legendRanges = (ranges: RangeSpec[]): BaseLegendProps["ranges"] =>
  ranges.map((r) => ({
    key: r.key,
    name: r.label ?? r.key,
    color: r.color,
    zero: r.zero,
    start:
      r.customZero === true && r.window.variant === "static"
        ? TimeStamp.z.parse(r.window.timeRange.start)
        : undefined,
  }));

const fetchingMessage = (ranges: RangeSpec[] | undefined): string | undefined => {
  if (ranges == null || ranges.length === 0) return undefined;
  const windows = ranges.map((r) => r.window);
  const now = TimeStamp.now();
  // A range that still runs counts the time it has run, and one that has not started
  // counts none.
  const past = new TimeRange(TimeStamp.ZERO, now);
  const spans = windows.map((w) => resolveWindow(w, now).boundBy(past).span);
  const longest = spans.reduce((a, b) => (b.greaterThan(a) ? b : a));
  if (longest.isZero) return undefined;
  return `Fetching ${longest.toString("semantic")} of data`;
};

export const LinePlot = ({
  editable = true,
  enableTriggers = true,
  ranges = [],
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
  hiddenRanges,
  hiddenTraces,
  onLineVisibleChange,
  onRangeVisibleChange,
  onRangeColorChange,
  onHoveredChange,
  relativeDisabled = false,
  children,
  ref,
  ...rest
}: LinePlotProps): ReactElement => {
  const key = useKey();
  useUndoRedoTriggers({ key, enabled: enableTriggers });
  const xAxisKeys = useXAxisKeys({ key });
  const x1 = useXAxis({ key, axisKey: "x1" });
  const relative = !relativeDisabled && isRelative(x1);
  const viewportRef = useViewportReset({ key, relative });
  const [hoverKey, setHoverKey] = useState<string | null>(null);
  const handleTooltipHover = (found: tooltip.Hovered | null): void => {
    const trace = found == null ? null : parseTraceKey(found.key, ranges);
    const zero = ranges.find((r) => r.key === trace?.range)?.zero;
    onHoveredChange?.(
      trace == null || zero == null || found == null
        ? null
        : { ...trace, time: zero.add(BigInt(Math.round(found.x))) },
    );
  };
  const hover = parseHover(hoverKey, {
    relative,
    ranges,
    hiddenLines,
    hiddenRanges,
    hiddenTraces,
  });
  const plotDropProps = usePlotDrop(key, editable);
  return (
    <Frame
      ref={ref}
      viewportRef={viewportRef}
      loadingMessage={fetchingMessage(ranges)}
      onViewportChange={onViewportChange}
      {...rest}
    >
      <HoverContext value={{ hover, hiddenRanges, hiddenTraces }}>
        {xAxisKeys.map((xAxisKey) => (
          <XAxis
            key={xAxisKey}
            pKey={key}
            axisKey={xAxisKey}
            editable={editable}
            ranges={ranges}
            hiddenLines={hiddenLines}
            onSelectRule={onSelectRule}
            rangeProviderProps={rangeProviderProps}
            relativeDisabled={relativeDisabled}
          />
        ))}
      </HoverContext>
      <Legend
        pKey={key}
        editable={editable}
        onLineVisibleChange={onLineVisibleChange}
        ranges={relative ? legendRanges(ranges) : undefined}
        hiddenRanges={hiddenRanges}
        onRangeVisibleChange={onRangeVisibleChange}
        onRangeColorChange={onRangeColorChange}
        onHoverChange={setHoverKey}
      />
      <Title pKey={key} editable={editable} />
      <BaseViewport
        initial={initialViewport}
        triggers={viewportTriggers}
        ref={viewportRef}
        {...plotDropProps}
      >
        {enableTooltip && (
          <Tooltip.Tooltip
            relative={relative}
            onHoveredChange={relative ? handleTooltipHover : undefined}
          />
        )}
        {enableMeasure && (
          <Measure.Measure
            mode={measureMode}
            onModeChange={onMeasureModeChange}
            relative={relative}
          />
        )}
        {children}
      </BaseViewport>
    </Frame>
  );
};

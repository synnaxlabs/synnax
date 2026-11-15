// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/tree/Timeline.css";

import { ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { context } from "@synnaxlabs/lyra/context";
import { Flex } from "@synnaxlabs/lyra/flex";
import {
  useCombinedRefs,
  useNow,
  useResize,
  useSyncedRef,
} from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Store } from "@synnaxlabs/lyra/store";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access } from "@synnaxlabs/pluto";
import { box, color, type TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type MouseEvent,
  type PointerEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";

import { Lead, scheduled } from "@/feature/range/tree/Item";
import { type Node, useNode } from "@/feature/range/tree/tree";
import { CSS } from "@/platform/css";
import { Range } from "@/platform/range";

/** A window of time in milliseconds since the epoch. */
interface Window {
  start: number;
  end: number;
}

const MS_PER_NS = 1e6;
const toMS = (ts: TimeStamp): number => Number(ts.valueOf()) / MS_PER_NS;
const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;
/** Steps shorter than a day, aligned to local time. */
const CLOCK_STEPS = [
  SECOND,
  5 * SECOND,
  15 * SECOND,
  30 * SECOND,
  MINUTE,
  5 * MINUTE,
  15 * MINUTE,
  30 * MINUTE,
  HOUR,
  3 * HOUR,
  6 * HOUR,
  12 * HOUR,
];

type Unit = "clock" | "day" | "week" | "month" | "quarter" | "year";

interface Step {
  unit: Unit;
  /** The step in milliseconds, exact for the clock and the day, typical otherwise. */
  size: number;
}

const STEPS: Step[] = [
  ...CLOCK_STEPS.map((size) => ({ unit: "clock" as const, size })),
  { unit: "day", size: DAY },
  { unit: "week", size: 7 * DAY },
  { unit: "month", size: 30 * DAY },
  { unit: "quarter", size: 91 * DAY },
  { unit: "year", size: 365 * DAY },
];

const MIN_TICK_PX = 84;
const MIN_SPAN = 2 * SECOND;
/** How many of the first top-level ranges the default window holds. */
const FIT_COUNT = 12;
/** How much one press of a zoom button widens or narrows the window. */
const ZOOM_STEP = 2;
/** Where now sits across the window after Now, so recent ranges show to its left. */
const NOW_AT = 0.8;

/** @returns the window of a scheduled time range. An open one runs to now. */
const span = (timeRange: TimeRange, now: number): Window => {
  const { start, end } = timeRange.makeValid();
  const s = toMS(start);
  const e = end.equals(TimeStamp.MAX) ? Math.max(now, s) : toMS(end);
  return { start: s, end: e };
};

const pad = ({ start, end }: Window, fraction = 0.04): Window => {
  const size = Math.max(end - start, MIN_SPAN);
  return { start: start - size * fraction, end: end + size * fraction };
};

/** @returns the window that holds every scheduled time range, or null for none. */
const fit = (timeRanges: TimeRange[], now: number): Window | null => {
  const spans = timeRanges.filter(scheduled).map((tr) => span(tr, now));
  if (spans.length === 0) return null;
  return pad({
    start: Math.min(...spans.map((s) => s.start)),
    end: Math.max(...spans.map((s) => s.end)),
  });
};

const offset = (ms: number): number => new Date(ms).getTimezoneOffset() * MINUTE;

interface Tick {
  at: number;
  unit: Unit;
  size: number;
}

/** @returns the first local calendar boundary of the unit at or after the time. */
const firstBoundary = (ms: number, unit: Unit): Date => {
  const d = new Date(ms);
  d.setHours(0, 0, 0, 0);
  if (unit === "week") d.setDate(d.getDate() - ((d.getDay() + 6) % 7));
  if (unit === "month" || unit === "quarter" || unit === "year") d.setDate(1);
  if (unit === "quarter") d.setMonth(d.getMonth() - (d.getMonth() % 3));
  if (unit === "year") d.setMonth(0);
  return d;
};

const advance = (d: Date, unit: Unit): void => {
  if (unit === "day") d.setDate(d.getDate() + 1);
  else if (unit === "week") d.setDate(d.getDate() + 7);
  else if (unit === "month") d.setMonth(d.getMonth() + 1);
  else if (unit === "quarter") d.setMonth(d.getMonth() + 3);
  else d.setFullYear(d.getFullYear() + 1);
};

/**
 * The ticks of the axis. Steps under a day align to local time, and longer ones to the
 * calendar: weeks start on Monday, months on the first.
 */
const ticks = (w: Window, width: number): Tick[] => {
  const span = w.end - w.start;
  const { unit, size } =
    STEPS.find((s) => (s.size / span) * width >= MIN_TICK_PX) ??
    STEPS[STEPS.length - 1];
  const out: Tick[] = [];
  if (unit === "clock") {
    const local = (ms: number) => ms - offset(ms);
    const first = Math.ceil(local(w.start) / size) * size;
    for (let t = first; t <= local(w.end) && out.length < 200; t += size)
      out.push({ at: t + offset(t), unit, size });
    return out;
  }
  const d = firstBoundary(w.start, unit);
  while (d.getTime() < w.start) advance(d, unit);
  for (; d.getTime() <= w.end && out.length < 200; advance(d, unit))
    out.push({ at: d.getTime(), unit, size });
  return out;
};

const formatTick = ({ at, unit, size }: Tick, first: boolean): string => {
  const d = new Date(at);
  if (unit === "year") return String(d.getFullYear());
  if (unit === "month" || unit === "quarter") {
    const month = d.toLocaleDateString(undefined, { month: "short" });
    return first || d.getMonth() === 0 ? `${month} ${d.getFullYear()}` : month;
  }
  const date = d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  if (unit !== "clock") return date;
  const time = d.toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: size < MINUTE ? "2-digit" : undefined,
    hour12: false,
  });
  const midnight = d.getHours() === 0 && d.getMinutes() === 0 && d.getSeconds() === 0;
  if (midnight) return date;
  return first ? `${date} ${time}` : time;
};

const GRID_LINE = "var(--console-range-timeline-line)";
const GRID_FILL = `linear-gradient(${GRID_LINE}, ${GRID_LINE})`;

/** One grid line per tick, as background layers, since months differ in length. */
const grid = (marks: Tick[], w: Window): string =>
  marks
    .map(({ at }) => `${GRID_FILL} ${percent(at, w)}% 0 / 1px 100% no-repeat`)
    .join(", ") || "none";

interface ContextValue {
  /** The time that bar positions count from, in milliseconds since the epoch. */
  origin: number;
  onZoom: (to: Window) => void;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tree.TimelineContext",
  providerName: "Tree.Timeline",
});

const percent = (ms: number, w: Window): number =>
  ((ms - w.start) / (w.end - w.start)) * 100;

interface RunningForProps {
  start: TimeStamp;
}

const RunningFor = ({ start }: RunningForProps): ReactElement => {
  const elapsed = Telem.Text.useTimeSpanSince(start);
  return (
    <Text.Text level="small" color={9}>
      Running for {elapsed.toString("semantic")}
    </Text.Text>
  );
};

interface BarProps {
  range: ranger.Range;
}

/** A bar places itself from the window on the timeline root, so a pan renders none. */
const Bar = ({ range }: BarProps): ReactElement | null => {
  const { origin, onZoom } = useContext("Tree.Bar");
  if (!scheduled(range.timeRange)) return null;
  const open = range.timeRange.end.equals(TimeStamp.MAX);
  const { start, end } = range.timeRange.makeValid();
  const hue = range.color == null ? undefined : color.hex(range.color);
  const style = {
    "--console-range-bar-start": toMS(start) - origin,
    ...(!open && { "--console-range-bar-end": toMS(end) - origin }),
    ...(hue != null && { "--console-range-color": hue }),
  } as CSSProperties;
  const handleClick = (e: MouseEvent) => {
    e.stopPropagation();
    onZoom(pad(span(range.timeRange, toMS(TimeStamp.now())), 0.1));
  };
  return (
    <Tooltip.Dialog location="top">
      <Flex.Box y gap="tiny">
        <Text.Text level="small" weight={450}>
          {range.name}
        </Text.Text>
        <Telem.Text.TimeRange level="small" color={9}>
          {range.timeRange}
        </Telem.Text.TimeRange>
        {open ? (
          <RunningFor start={start} />
        ) : (
          <Text.Text level="small" color={9}>
            {start.span(end).toString("semantic")}
          </Text.Text>
        )}
      </Flex.Box>
      <div
        className={CSS.cls(
          CSS.BE("range-timeline", "bar"),
          hue == null && CSS.BEM("range-timeline", "bar", "plain"),
          open && CSS.BEM("range-timeline", "bar", "open"),
        )}
        style={style}
        onClick={handleClick}
        aria-label={`Zoom to ${range.name}`}
      />
    </Tooltip.Dialog>
  );
};

/**
 * One range of a timeline: its caret, color, and name at its depth in the tree, then
 * its bar over the shared axis.
 */
const Lane = ({
  className,
  ...rest
}: List.ItemProps<ranger.Key>): ReactElement | null => {
  const { itemKey } = rest;
  const range = List.useItem<ranger.Key, ranger.Range>(itemKey);
  const { depth, expanded, onExpand } = useNode(itemKey);
  const preview = !Access.useUpdateGranted(ranger.resourceID(itemKey));
  const open = Range.useOpen();
  if (range == null) return null;
  return (
    <Select.Item<ranger.Key>
      {...rest}
      role="listitem"
      className={CSS.cls(CSS.BE("range-timeline", "lane"), className)}
      checkable
      onActivate={() => open(itemKey)}
      aria-label={range.name}
    >
      <Flex.Box x align="center" className={CSS.BE("range-timeline", "label")}>
        <Lead
          range={range}
          depth={depth}
          expanded={expanded}
          onExpand={onExpand}
          preview={preview}
        />
      </Flex.Box>
      <div className={CSS.BE("range-timeline", "track")}>
        <Bar range={range} />
      </div>
    </Select.Item>
  );
};

export interface TimelineProps {
  nodes: Node[];
  /** A time range that the window holds beside the ranges, such as a parent's. */
  bounds?: TimeRange;
  emptyContent?: ReactNode;
  /** Shows after the last lane. */
  footer?: ReactNode;
  onContextMenu?: List.ScrollProps["onContextMenu"];
}

const nextTick = (now: TimeStamp) => now.add(TimeSpan.seconds(15));

const lane = Component.renderProp(Lane);

interface FrameProps extends Omit<TimelineProps, "emptyContent"> {
  lanes: ReactNode;
}

/** Holds the window. The lanes come in built, so a pan renders the axis and no lane. */
const Frame = ({
  nodes,
  bounds,
  lanes,
  footer,
  onContextMenu,
}: FrameProps): ReactElement => {
  const now = toMS(useNow(nextTick));
  const { getItem, subscribe } = List.useUtilContext<ranger.Key, ranger.Range>();
  if (getItem == null || subscribe == null)
    throw new Error("Tree.Timeline must be inside a list of ranges");
  const roots = Store.useItems({
    keys: nodes
      .filter((n) => n.depth === 0)
      .slice(0, FIT_COUNT)
      .map((n) => n.key),
    subscribe,
    getItem,
  });
  const fitted = fit(
    [...(bounds == null ? [] : [bounds]), ...roots.map((r) => r.timeRange)],
    now,
  );
  const [view, setView] = useState<Window | null>(null);
  const [ctx] = useState<ContextValue>(() => ({ origin: now, onZoom: setView }));
  const w = view ?? fitted ?? { start: now - DAY, end: now };
  const axisRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(800);
  const resizeRef = useResize<HTMLDivElement>((b) => setWidth(box.width(b)));
  const axisRefs = useCombinedRefs(axisRef, resizeRef);
  const scrollRef = useRef<HTMLDivElement>(null);
  const geometry = () => axisRef.current?.getBoundingClientRect();
  /** Zooms by the factor and holds the time at the same place on the axis. */
  const zoomAround = (at: number, factor: number) => {
    const x = (at - w.start) / (w.end - w.start);
    const size = Math.max((w.end - w.start) * factor, MIN_SPAN);
    setView({ start: at - x * size, end: at + (1 - x) * size });
  };
  const zoomAt = (clientX: number, factor: number) => {
    const rect = geometry();
    if (rect == null) return;
    const x = Math.min(Math.max((clientX - rect.left) / rect.width, 0), 1);
    zoomAround(w.start + x * (w.end - w.start), factor);
  };
  /** Zooms around now when it shows, and around the middle when it does not. */
  const zoomBy = (factor: number) =>
    zoomAround(now >= w.start && now <= w.end ? now : (w.start + w.end) / 2, factor);
  const showNow = () => {
    const size = w.end - w.start;
    setView({ start: now - size * NOW_AT, end: now + size * (1 - NOW_AT) });
  };
  const panBy = (px: number) => {
    const rect = geometry();
    if (rect == null) return;
    const shift = (px / rect.width) * (w.end - w.start);
    setView({ start: w.start + shift, end: w.end + shift });
  };
  const wheelRef = useSyncedRef((e: WheelEvent) => {
    const rect = geometry();
    if (rect == null || e.clientX < rect.left) return;
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      zoomAt(e.clientX, Math.exp(e.deltaY * 0.01));
    } else if (Math.abs(e.deltaX) > Math.abs(e.deltaY) || e.shiftKey) {
      e.preventDefault();
      panBy(e.shiftKey ? e.deltaY : e.deltaX);
    }
  });
  useEffect(() => {
    const el = scrollRef.current;
    if (el == null) return;
    const handleWheel = (e: WheelEvent) => wheelRef.current(e);
    el.addEventListener("wheel", handleWheel, { passive: false });
    return () => el.removeEventListener("wheel", handleWheel);
  }, [wheelRef]);
  const drag = useRef<number | null>(null);
  const handlePointerDown = (e: PointerEvent<HTMLDivElement>) => {
    drag.current = e.clientX;
    e.currentTarget.setPointerCapture(e.pointerId);
  };
  const handlePointerMove = (e: PointerEvent<HTMLDivElement>) => {
    if (drag.current == null) return;
    panBy(drag.current - e.clientX);
    drag.current = e.clientX;
  };
  const handlePointerUp = () => {
    drag.current = null;
  };
  const marks = ticks(w, width);
  const style = {
    "--console-range-timeline-start": w.start - ctx.origin,
    "--console-range-timeline-end": w.end - ctx.origin,
    "--console-range-timeline-now": now - ctx.origin,
    "--console-range-timeline-grid": grid(marks, w),
  } as CSSProperties;
  return (
    <Context value={ctx}>
      <List.Scroll
        ref={scrollRef}
        grow
        className={CSS.B("range-timeline")}
        style={style}
        onContextMenu={onContextMenu}
      >
        <div className={CSS.BE("range-timeline", "header")}>
          <Flex.Box x align="center" className={CSS.BE("range-timeline", "label")}>
            <Flex.Box
              x
              align="center"
              gap="tiny"
              className={CSS.BE("range-timeline", "controls")}
            >
              <Button.Button
                variant="text"
                size="small"
                textColor={9}
                tooltip="Zoom out"
                tooltipLocation="bottom"
                onClick={() => zoomBy(ZOOM_STEP)}
              >
                <Icon.Subtract />
              </Button.Button>
              <Button.Button
                variant="text"
                size="small"
                textColor={9}
                tooltip="Zoom in"
                tooltipLocation="bottom"
                onClick={() => zoomBy(1 / ZOOM_STEP)}
              >
                <Icon.Add />
              </Button.Button>
              <Button.Button
                variant="text"
                size="small"
                textColor={9}
                onClick={showNow}
              >
                Now
              </Button.Button>
              {view != null && (
                <Button.Button
                  variant="text"
                  size="small"
                  textColor={9}
                  onClick={() => setView(null)}
                >
                  Fit all
                </Button.Button>
              )}
            </Flex.Box>
          </Flex.Box>
          <div
            ref={axisRefs}
            className={CSS.BE("range-timeline", "axis")}
            role="group"
            aria-label="Time axis"
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
          >
            {marks.map((mark, i) => {
              // A label too near the end would clip, so its tick shows only a line.
              const x = (percent(mark.at, w) / 100) * width;
              if (x + MIN_TICK_PX > width) return null;
              return (
                <Text.Text
                  key={mark.at}
                  level="tiny"
                  color={9}
                  className={CSS.BE("range-timeline", "tick")}
                  style={
                    {
                      "--console-range-timeline-tick": mark.at - ctx.origin,
                    } as CSSProperties
                  }
                >
                  {formatTick(mark, i === 0)}
                </Text.Text>
              );
            })}
          </div>
        </div>
        {lanes}
        {footer}
      </List.Scroll>
    </Context>
  );
};

/**
 * Lays ranges out as lanes over one time axis. Pinch or Cmd and scroll to zoom, scroll
 * sideways or drag the axis to pan, and click a bar to zoom to it.
 */
export const Timeline = ({ emptyContent, ...rest }: TimelineProps): ReactElement => (
  <Frame
    {...rest}
    lanes={
      rest.nodes.length === 0 ? (
        emptyContent
      ) : (
        <List.Items<ranger.Key>>{lane}</List.Items>
      )
    }
  />
);

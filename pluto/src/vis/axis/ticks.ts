// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type scale, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ScaleLinear, scaleLinear, type ScaleTime, scaleTime } from "d3-scale";
import { timeTickInterval } from "d3-time";
import { z } from "zod";

import { type PreciseTimeScale, preciseTimeScale } from "@/vis/axis/preciseTimeScale";
import { formatRelativeTime, RELATIVE_STEPS } from "@/vis/axis/relative";

export interface Tick {
  position: number;
  label: string;
  /** The index of the segment that holds the tick, on an axis with breaks. */
  segment?: number;
  /**
   * The label of the tick when it anchors its segment: the full date and time. The
   * first tick of each segment whose anchor label fits is the anchor.
   */
  anchorLabel?: string;
}

/** A segment of an axis with breaks, which maps values with its own scale. */
export interface TickSegment {
  /** The decimal positions across the axis that the segment fills. */
  position: bounds.Bounds;
  /** Takes a decimal position across the axis and returns the value there. */
  decimalToDataScale: scale.Scale;
}

export interface TickFactory {
  create: (params: TickFactoryRenderParams) => Tick[];
}

export const tickType = z.enum(["linear", "time", "relative"]);

export type TickType = z.infer<typeof tickType>;

export const tickFactoryProps = z.object({
  tickSpacing: z.number().default(75),
  type: tickType.default("linear"),
});

export type TickFactoryProps = z.input<typeof tickFactoryProps>;
type ParsedTickFactoryProps = z.infer<typeof tickFactoryProps>;

export interface TickFactoryRenderParams {
  /** Scale takes a value in decimal space and returns the corresponding data value. */
  decimalToDataScale: scale.Scale;
  /** Size is the length of the axis in pixels. */
  size: number;
  /**
   * The segments of an axis with breaks, in order. Only a time axis takes more than
   * one. The segments share one tick interval, and each tick has an anchor label.
   */
  segments?: TickSegment[];
}

export const newTickFactory = (props: TickFactoryProps): TickFactory => {
  const parsed = tickFactoryProps.parse(props);
  return TICK_FACTORIES[parsed.type](parsed);
};

class TimeTickFactory implements TickFactory {
  private readonly props: ParsedTickFactoryProps;

  private prevScaleSize: number;
  private prevDomain: TimeRange;
  private currTicks: Tick[];

  private readonly normalScale: ScaleTime<number, number>;
  private readonly preciseScale: PreciseTimeScale;

  constructor(props: ParsedTickFactoryProps) {
    this.props = props;
    this.normalScale = scaleTime();
    this.preciseScale = preciseTimeScale();
    this.prevScaleSize = 0;
    this.prevDomain = new TimeRange(new TimeStamp(0), new TimeStamp(0));
    this.currTicks = [];
  }

  create({
    decimalToDataScale: scale,
    size,
    segments = [],
  }: TickFactoryRenderParams): Tick[] {
    const visible = segments.filter(
      ({ position: { lower, upper } }) => Math.max(lower, 0) < Math.min(upper, 1),
    );
    // With one segment in view, no seam is in view, so the axis ticks as one with no
    // breaks.
    if (visible.length > 1) return this.segmentTicks(visible, size);
    if (visible.length === 1) scale = visible[0].decimalToDataScale;
    const domain = new TimeRange(
      new TimeStamp(scale.pos(0)),
      new TimeStamp(scale.pos(1)),
    );
    if (this.prevDomain.equals(domain) && this.prevScaleSize === size)
      return this.currTicks;
    this.prevDomain = domain;

    if (this.prevScaleSize !== size) {
      const range: [number, number] = [0, size];
      this.preciseScale.range(range);
      this.normalScale.range(range);
      this.prevScaleSize = size;
    }

    if (domain.span.milliseconds < 5) {
      this.preciseScale.domain([domain.start, domain.end]);
      const count = calcTickCount(size, this.props.tickSpacing) / 2;
      const ticks = this.preciseScale.ticks(count);
      this.currTicks = ticks.map((tick) => ({
        label: this.preciseScale.formatTick(tick),
        position: this.preciseScale.scale(tick),
      }));
    } else {
      this.normalScale.domain([domain.start.date(), domain.end.date()]);
      const count = calcTickCount(size, this.props.tickSpacing);
      let ticks = this.normalScale.ticks(count);
      // Re-request fewer ticks when d3 overshoots the count and crowds the labels.
      if (ticks.length > count)
        ticks = this.normalScale.ticks(Math.ceil(count ** 2 / ticks.length));
      ticks = evenlySpaceTwoDayTicks(ticks, domain);
      this.currTicks = ticks.map((tick) => ({
        label: this.normalTickLabel(tick),
        position: this.normalScale(tick),
      }));
    }
    return this.currTicks;
  }

  // segmentTicks places ticks in the visible part of each segment at one interval,
  // which the visible time of all segments and the size of the axis pick.
  private segmentTicks(segments: TickSegment[], size: number): Tick[] {
    const visible = segments.map(({ position, decimalToDataScale: s }) => ({
      domain: new TimeRange(
        new TimeStamp(s.pos(Math.max(position.lower, 0))),
        new TimeStamp(s.pos(Math.min(position.upper, 1))),
      ),
      toDecimal: s.reverse(),
    }));
    const total = visible.reduce(
      (sum, { domain }) => sum + domain.span.milliseconds,
      0,
    );
    const count = calcTickCount(size, this.props.tickSpacing);
    const interval = timeTickInterval(new Date(0), new Date(total), count);
    if (interval == null) return [];
    return visible.flatMap(({ domain, toDecimal }, segment) => {
      const range = interval.range(domain.start.date(), domain.end.date());
      const ticks = evenlySpaceTwoDayTicks(range, domain);
      return ticks.map((tick) => ({
        label: this.normalTickLabel(tick),
        anchorLabel: anchorLabel(tick),
        segment,
        position: toDecimal.pos(Number(new TimeStamp(tick).valueOf())) * size,
      }));
    });
  }

  normalTickLabel(date: Date): string {
    const ms = date.getMilliseconds();
    if (date.getSeconds() !== 0 || ms !== 0) {
      let formatted: string = `${date.getSeconds()}`;
      if (ms !== 0) {
        const millisecondString = Math.round(ms)
          .toString()
          .padStart(3, "0")
          .replace(/0+$/, "");
        formatted += `.${millisecondString}`;
      }
      return `${formatted}s`;
    }
    // If we're on the minute, show the hour and minute in military time
    if (date.getHours() !== 0 || date.getMinutes() !== 0)
      return `${date.getHours()}:${date.getMinutes().toString().padStart(2, "0")}`;
    // Midnight ticks label the finest calendar unit that changed instead.
    const month = MONTH_FORMAT.format(date);
    if (date.getDate() !== 1) return `${month} ${date.getDate()}`;
    if (date.getMonth() !== 0) return month;
    return `${date.getFullYear()}`;
  }
}

class LinearTickFactory implements TickFactory {
  private readonly props: ParsedTickFactoryProps;
  private prevDomain: bounds.Bounds;
  private prevScaleSize: number;
  private currTicks: Tick[];
  private d3Scale: ScaleLinear<number, number>;

  constructor(props: ParsedTickFactoryProps) {
    this.props = props;
    this.prevDomain = bounds.construct(0, 0);
    this.prevScaleSize = 0;
    this.currTicks = [];
    this.d3Scale = scaleLinear();
  }

  create({ decimalToDataScale: scale, size }: TickFactoryRenderParams): Tick[] {
    const domain = { lower: scale.pos(0), upper: scale.pos(1) };
    if (bounds.equals(this.prevDomain, domain) && this.prevScaleSize === size)
      return this.currTicks;
    if (!bounds.equals(this.prevDomain, domain))
      this.d3Scale = this.d3Scale.domain([domain.lower, domain.upper]);
    if (this.prevScaleSize !== size) this.d3Scale = this.d3Scale.range([0, size]);
    this.prevDomain = domain;
    this.prevScaleSize = size;

    const count = calcTickCount(size, this.props.tickSpacing);
    const ticks = this.d3Scale.ticks(count);
    this.currTicks = ticks.map((tick) => ({
      label: this.tickLabel(tick),
      position: this.d3Scale(tick),
    }));
    return this.currTicks;
  }

  tickLabel(value: number): string {
    return value.toString();
  }
}

// RelativeTickFactory labels a span in nanoseconds from a zero, such as the time since
// the start of a range: T-5 s, T0, T+5 s.
class RelativeTickFactory implements TickFactory {
  private readonly props: ParsedTickFactoryProps;

  constructor(props: ParsedTickFactoryProps) {
    this.props = props;
  }

  create({ decimalToDataScale: scale, size }: TickFactoryRenderParams): Tick[] {
    const lower = scale.pos(0);
    const upper = scale.pos(1);
    const span = upper - lower;
    if (!(span > 0) || size <= 0) return [];
    const target = span / calcTickCount(size, this.props.tickSpacing);
    const step =
      RELATIVE_STEPS.find((s) => s >= target) ??
      RELATIVE_STEPS[RELATIVE_STEPS.length - 1];
    const magnitude = Math.max(Math.abs(lower), Math.abs(upper));
    const ticks: Tick[] = [];
    for (let v = Math.ceil(lower / step) * step; v <= upper; v += step)
      ticks.push({
        label: formatRelativeTime(v, step, magnitude),
        position: ((v - lower) / span) * size,
      });
    return ticks;
  }
}

// anchorLabel labels a date with its day and its time, to the finest unit that is not
// zero: "Nov 18 10:00", "Nov 18 10:00:30", "Nov 18 10:00:30.25".
const anchorLabel = (date: Date): string => {
  const time = `${date.getHours()}:${date.getMinutes().toString().padStart(2, "0")}`;
  const ms = date.getMilliseconds();
  let seconds = "";
  if (date.getSeconds() !== 0 || ms !== 0)
    seconds = `:${date.getSeconds().toString().padStart(2, "0")}`;
  if (ms !== 0) seconds += `.${ms.toString().padStart(3, "0").replace(/0+$/, "")}`;
  return `${MONTH_FORMAT.format(date)} ${date.getDate()} ${time}${seconds}`;
};

const calcTickCount = (size: number, pixelsPerTick: number): number => {
  const tickCount = Math.floor(size / pixelsPerTick);
  return tickCount > 0 ? tickCount : 1;
};

const DAY_MS = TimeSpan.DAY.milliseconds;

const MONTH_FORMAT = new Intl.DateTimeFormat("default", { month: "short" });

const isLocalMidnight = (date: Date): boolean =>
  date.getHours() === 0 &&
  date.getMinutes() === 0 &&
  date.getSeconds() === 0 &&
  date.getMilliseconds() === 0;

const localEpochDay = (date: Date): number =>
  Math.floor((date.getTime() - date.getTimezoneOffset() * 60_000) / DAY_MS);

// d3's two-day tick interval restarts on the first of each month, so ticks at a month
// boundary can sit one day apart. Rebuild them on a two-day grid anchored to the epoch.
const evenlySpaceTwoDayTicks = (ticks: Date[], domain: TimeRange): Date[] => {
  if (ticks.length < 2 || !ticks.every(isLocalMidnight)) return ticks;
  const gapDays = ticks
    .slice(1)
    .map((tick, i) => Math.round((tick.getTime() - ticks[i].getTime()) / DAY_MS));
  if (!gapDays.every((gap) => gap === 1 || gap === 2) || !gapDays.includes(2))
    return ticks;
  const cursor = new Date(domain.start.milliseconds);
  cursor.setHours(0, 0, 0, 0);
  if (cursor.getTime() < domain.start.milliseconds)
    cursor.setDate(cursor.getDate() + 1);
  if (localEpochDay(cursor) % 2 !== 0) cursor.setDate(cursor.getDate() + 1);
  const end = domain.end.milliseconds;
  const rebuilt: Date[] = [];
  while (cursor.getTime() <= end) {
    rebuilt.push(new Date(cursor));
    cursor.setDate(cursor.getDate() + 2);
  }
  return rebuilt;
};

const TICK_FACTORIES: Record<TickType, (props: ParsedTickFactoryProps) => TickFactory> =
  {
    linear: (p) => new LinearTickFactory(p),
    time: (p) => new TimeTickFactory(p),
    relative: (p) => new RelativeTickFactory(p),
  };

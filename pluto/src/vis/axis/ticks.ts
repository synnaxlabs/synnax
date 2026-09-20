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
import { z } from "zod";

import { type PreciseTimeScale, preciseTimeScale } from "@/vis/axis/preciseTimeScale";

export interface Tick {
  position: number;
  label: string;
}

export interface TickFactory {
  create: (params: TickFactoryRenderParams) => Tick[];
}

export const tickType = z.enum(["linear", "time"]);

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

  create({ decimalToDataScale: scale, size }: TickFactoryRenderParams): Tick[] {
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
  };

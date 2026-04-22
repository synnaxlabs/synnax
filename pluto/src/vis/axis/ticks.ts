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
  create: (ctx: TickFactoryRenderArgs) => Tick[];
}

export const tickType = z.enum(["linear", "time", "relativeTime"]);

export type TickType = z.infer<typeof tickType>;

export const tickFactoryProps = z.object({
  tickSpacing: z.number().default(75),
  type: tickType.default("linear"),
});

export type TickFactoryProps = z.input<typeof tickFactoryProps>;
type ParsedTickFactoryProps = z.infer<typeof tickFactoryProps>;

export interface TickFactoryRenderArgs {
  /**
   * Scale takes a value in decimal space and returns the corresponding data value.
   */
  decimalToDataScale: scale.Scale;
  /**
   * Size is the length of the axis in pixels.
   */
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

  create({ decimalToDataScale: scale, size }: TickFactoryRenderArgs): Tick[] {
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
      const ticks = this.normalScale.ticks(calcTickCount(size, this.props.tickSpacing));
      this.currTicks = ticks.map((tick) => ({
        label: this.normalTickLabel(tick),
        position: this.normalScale(tick),
      }));
    }
    return this.currTicks;
  }

  normalTickLabel(date: Date): string {
    let formatted: string = `:${date.getSeconds()}`;
    const ms = date.getMilliseconds();
    if (ms !== 0) {
      const millisecondString = Math.round(date.getMilliseconds())
        .toString()
        .padStart(3, "0")
        .replace(/0+$/, "");
      formatted += `.${millisecondString}`;
    }
    // If we're on the minute, show the hour and minute in military time
    if (date.getSeconds() === 0 && ms === 0)
      formatted = `${date.getHours()}:${date.getMinutes().toString().padStart(2, "0")}`;
    return formatted;
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

  create({ decimalToDataScale: scale, size }: TickFactoryRenderArgs): Tick[] {
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

export const formatRelativeTime = (nanos: number): string => {
  if (!Number.isFinite(nanos)) return "0s";
  const rounded = Math.round(Math.abs(nanos));
  if (!Number.isSafeInteger(rounded)) return "0s";
  const span = new TimeSpan(rounded);
  const prefix = nanos < 0 ? "-" : "";
  const totalMs = span.milliseconds;
  if (totalMs < 1) {
    const us = Math.round(span.microseconds);
    return `${prefix}${us}µs`;
  }
  const totalSec = Math.floor(span.seconds);
  const ms = Math.round(totalMs - totalSec * 1000);
  const totalMin = Math.floor(span.minutes);
  const sec = totalSec - totalMin * 60;
  const totalHr = Math.floor(span.hours);
  const min = totalMin - totalHr * 60;
  let str = "";
  if (totalHr > 0) str += `${totalHr}:${min.toString().padStart(2, "0")}:`;
  else if (totalMin > 0) str += `${min}:`;
  else {
    if (ms > 0)
      return `${prefix}${sec}.${ms.toString().padStart(3, "0").replace(/0+$/, "")}s`;
    return `${prefix}${sec}s`;
  }
  str += sec.toString().padStart(2, "0");
  if (ms > 0) str += `.${ms.toString().padStart(3, "0").replace(/0+$/, "")}`;
  return prefix + str;
};

class RelativeTimeTickFactory implements TickFactory {
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

  create({ decimalToDataScale: scale, size }: TickFactoryRenderArgs): Tick[] {
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
      label: formatRelativeTime(tick),
      position: this.d3Scale(tick),
    }));
    return this.currTicks;
  }
}

const calcTickCount = (size: number, pixelsPerTick: number): number => {
  const tickCount = Math.floor(size / pixelsPerTick);
  return tickCount > 0 ? tickCount : 1;
};

const TICK_FACTORIES: Record<TickType, (props: ParsedTickFactoryProps) => TickFactory> =
  {
    linear: (p) => new LinearTickFactory(p),
    time: (p) => new TimeTickFactory(p),
    relativeTime: (p) => new RelativeTimeTickFactory(p),
  };

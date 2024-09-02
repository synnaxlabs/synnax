// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type scale, TimeStamp } from "@synnaxlabs/x";
import { scaleLinear, scaleTime } from "d3-scale";
import { z } from "zod";

export interface Tick {
  position: number;
  label: string;
}

export interface TickFactory {
  generate: (ctx: TickFactoryContext) => Tick[];
}

export const tickType = z.enum(["linear", "time"]);

export type TickType = z.infer<typeof tickType>;

export const tickFactoryProps = z.object({
  tickSpacing: z.number().default(75),
  type: tickType.optional().default("linear"),
});

export type TickFactoryProps = z.input<typeof tickFactoryProps>;
type ParsedTickFactoryProps = z.output<typeof tickFactoryProps>;

export interface TickFactoryContext {
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

  constructor(props: ParsedTickFactoryProps) {
    this.props = props;
  }

  generate({ decimalToDataScale: scale, size }: TickFactoryContext): Tick[] {
    const range = [0, size];
    const domain = [
      new TimeStamp(scale.pos(0)).date(),
      new TimeStamp(scale.pos(1)).date(),
    ];
    const d3Scale = scaleTime().domain(domain).range(range);
    const ticks = d3Scale.ticks(calcTickCount(size, this.props.tickSpacing));
    return ticks.map((tick) => ({
      label: this.tickLabel(tick),
      position: d3Scale(tick),
    }));
  }

  tickLabel(date: Date): string {
    const value = new TimeStamp(date).date();
    // remove trailing 0s

    let formatted: string = `:${value.getSeconds()}`;
    const ms = value.getMilliseconds();
    if (ms !== 0) {
      const millisecondString = Math.round(value.getMilliseconds())
        .toString()
        .padStart(3, "0")
        .replace(/0+$/, "");
      formatted += `.${millisecondString}`;
    }
    // If we're on the minute, show the hour and minute in military time
    if (value.getSeconds() === 0 && value.getMilliseconds() === 0)
      formatted = `${value.getHours()}:${value
        .getMinutes()
        .toString()
        .padStart(2, "0")}`;
    return formatted;
  }
}

class LinearTickFactory implements TickFactory {
  private readonly props: ParsedTickFactoryProps;

  constructor(props: ParsedTickFactoryProps) {
    this.props = props;
  }

  generate({ decimalToDataScale: scale, size }: TickFactoryContext): Tick[] {
    const range = [0, size];
    const domain = [scale.pos(0), scale.pos(1)];
    const d3Scale = scaleLinear().domain(domain).range(range);
    const count = calcTickCount(size, this.props.tickSpacing);
    const ticks = d3Scale.ticks(count);
    return ticks.map((tick) => ({
      label: this.tickLabel(tick),
      position: d3Scale(tick),
    }));
  }

  tickLabel(value: number): string {
    return value.toString();
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
  };

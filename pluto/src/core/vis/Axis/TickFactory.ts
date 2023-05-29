import { OuterLocation, Scale, TimeStamp } from "@synnaxlabs/x";
import { scaleLinear, scaleTime } from "d3";

export interface Tick {
  position: number;
  label: string;
}

export interface TickFactory {
  generate: (ctx: TickFactoryContext) => Tick[];
}

export type TickType = "linear" | "time";

export interface TickFactoryProps {
  location: OuterLocation;
  tickSpacing: number;
  type: TickType;
}

export interface TickFactoryContext {
  /**
   * Scale takes a value in decimal space and returns the corresponding data value.
   */
  scale: Scale;
  /**
   * Size is the length of the axis in pixels.
   */
  size: number;
}

export const newTickFactory = (props: TickFactoryProps): TickFactory =>
  TICK_FACTORIES[props.type](props);

class TimeTickFactory implements TickFactory {
  private readonly props: TickFactoryProps;

  constructor(props: TickFactoryProps) {
    this.props = props;
  }

  generate({ scale, size }: TickFactoryContext): Tick[] {
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
  private readonly props: TickFactoryProps;

  constructor(props: TickFactoryProps) {
    this.props = props;
  }

  generate({ scale, size }: TickFactoryContext): Tick[] {
    const range = [0, size];
    const domain = [scale.pos(0), scale.pos(1)];
    const d3Scale = scaleLinear().domain(domain).range(range);
    const ticks = d3Scale.ticks(calcTickCount(size, this.props.tickSpacing));
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

const TICK_FACTORIES: Record<TickType, (props: TickFactoryProps) => TickFactory> = {
  linear: (p) => new LinearTickFactory(p),
  time: (p) => new TimeTickFactory(p),
};

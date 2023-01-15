import { useMemo } from "react";

import clsx from "clsx";
import * as d3 from "d3";

import { fRotate, fTranslate, locationRotations } from "../util/svg";

import { OuterLocation, XY, ZERO_XY } from "@/spatial";

import "./Axis.css";

import { TimeStamp } from "@synnaxlabs/x";

export type AxisType = "linear" | "time";

export interface AxisProps {
  range: [number, number];
  size: number;
  location?: OuterLocation;
  position?: XY;
  pixelsPerTick?: number;
  type?: AxisType;
  showGrid?: boolean;
  height?: number;
}

const calcTickCount = (size: number, pixelsPerTick: number): number =>
  Math.floor(size / pixelsPerTick);

export const Axis = ({
  range,
  size,
  position = ZERO_XY,
  location = "left",
  type = "linear",
  pixelsPerTick = 30,
  showGrid = false,
  height = 0,
}: AxisProps): JSX.Element => {
  const ticks: TickProps[] = useMemo(() => {
    const scale = initD3Scale(type, range).range(calcD3Range(location, size)) as Scale;
    const tickCount = calcTickCount(size, pixelsPerTick);
    return scale
      .ticks(tickCount)
      .map((value) => ({ value, offset: scale(value), location, showGrid, height }));
  }, [location, range, size, type]);

  const transform = calcGroupTransform(location, position, size);

  return (
    <g transform={transform} className="pluto-axis">
      <line x2={size} />
      {ticks.map((props) => (
        // eslint-disable-next-line react/prop-types
        <Tick key={props.value.valueOf()} {...props} />
      ))}
    </g>
  );
};

interface TickProps {
  value: number | Date;
  offset: number;
  location: OuterLocation;
  showGrid: boolean;
  height: number;
}

const tickYOffset = 4;

const Tick = ({ value, offset, showGrid, height }: TickProps): JSX.Element => (
  <g transform={fTranslate({ x: offset, y: -tickYOffset })}>
    {!showGrid ? <line y2={tickYOffset} /> : <line y2={height} />}
    {value instanceof Date ? (
      <DateTickText value={value} />
    ) : (
      <NumberTickText value={value} />
    )}
  </g>
);

const DateTickText = ({ value }: { value: Date }): JSX.Element => {
  let formatted: string = `:${value.getSeconds().toString().padEnd(2, "0")}`;
  // If we're on the minute, show the hour and minute in military time
  if (value.getSeconds() === 0)
    formatted = `${value.getHours()}:${value.getMinutes().toString().padStart(2, "0")}`;

  return <text transform={calcTickTextTranslate(formatted)}>{formatted}</text>;
};

const NumberTickText = ({ value }: { value: number }): JSX.Element => (
  <text transform={calcTickTextTranslate(value.toString())}>{value}</text>
);

const calcTickTextTranslate = (value: string): string =>
  clsx(fTranslate({ x: value.length * 3, y: -tickYOffset * 2.5 }), fRotate(180));

type Scale =
  | d3.ScaleLinear<number, number, never>
  | d3.ScaleTime<number, number, never>;

const initD3Scale = (type: AxisType, domain: [number, number]): Scale => {
  switch (type) {
    case "time":
      return d3.scaleTime().domain(domain.map((v) => new TimeStamp(v).date()));
    default:
      return d3.scaleLinear().domain(domain);
  }
};

const calcD3Range = (location: OuterLocation, size: number): [number, number] => {
  const base: [number, number] = [0, size];
  const reverse = ["bottom", "right"].includes(location);
  if (reverse) base.reverse();
  return base;
};

const calcGroupTransform = (
  location: OuterLocation,
  position: XY,
  size: number
): string => {
  const rotation = locationRotations[location];
  const adjustedPosition = { ...position };
  if (location === "left") adjustedPosition.y += size;
  else if (location === "bottom") adjustedPosition.x += size;
  return clsx(fTranslate(adjustedPosition), fRotate(rotation));
};

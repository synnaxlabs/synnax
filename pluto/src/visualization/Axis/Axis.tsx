import { useMemo } from "react";

import { TimeStamp } from "@synnaxlabs/x";
import clsx from "clsx";

import { fRotate, fTranslate, locationRotations } from "../util/svg";

import { OuterLocation, XY, ZERO_XY, Scale } from "@/spatial";

import "./Axis.css";

export type AxisType = "linear" | "time";

export interface AxisProps {
  scale: Scale;
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
  scale,
  size,
  position = ZERO_XY,
  location = "left",
  type = "linear",
  pixelsPerTick = 30,
  showGrid = false,
  height = 0,
}: AxisProps): JSX.Element => {
  const ticks: TickProps[] = useMemo(() => {
    let pxScale = scale.scale(size).reverse();
    if (["right", "bottom"].includes(location)) pxScale = pxScale.invert();
    const tickCount = calcTickCount(size, pixelsPerTick);
    const tickScale = Scale.scale(tickCount).scale(size);
    return Array.from({ length: tickCount }, (_, i) => {
      const offset = tickScale.pos(i);
      const value = pxScale.pos(offset);
      return { value: Math.floor(value), offset, location, showGrid, height, type };
    });
  }, [location, scale, size, type]);

  const transform = calcGroupTransform(location, position, size);

  return (
    <g transform={transform} className="pluto-axis">
      <line x2={size} />
      {ticks.map((props: TickProps) => (
        <Tick key={props.offset} {...props} />
      ))}
    </g>
  );
};

interface TickProps {
  value: number;
  offset: number;
  location: OuterLocation;
  showGrid: boolean;
  height: number;
  type: AxisType;
}

const tickYOffset = 4;

const Tick = ({ value, offset, showGrid, height, type }: TickProps): JSX.Element => (
  <g transform={fTranslate({ x: offset, y: -tickYOffset })}>
    {!showGrid ? <line y2={tickYOffset} /> : <line y2={height + tickYOffset} />}
    {type === "time" ? (
      <DateTickText value={value} />
    ) : (
      <NumberTickText value={value} />
    )}
  </g>
);

const DateTickText = ({ value: _value }: { value: number }): JSX.Element => {
  const value = new TimeStamp(_value).date();
  let formatted: string = `:${value.getSeconds()}`;
  // reverse the string

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

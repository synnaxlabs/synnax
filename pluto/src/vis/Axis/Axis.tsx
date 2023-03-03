// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemo } from "react";

import { TimeStamp, OuterLocation, XY, ZERO_XY, Scale } from "@synnaxlabs/x";
import clsx from "clsx";
import { scaleLinear, scaleTime } from "d3";

import { fRotate, fTranslate, locationRotations } from "../util/svg";

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

const timeTicks = (
  size: number,
  location: OuterLocation,
  scale: Scale,
  showGrid: boolean,
  height: number,
  count: number
): TickProps[] => {
  let range = [0, size];
  if (["bottom", "right"].includes(location)) range = range.reverse();
  const d3Scale = scaleTime()
    .domain([new TimeStamp(scale.pos(0)).date(), new TimeStamp(scale.pos(size)).date()])
    .range(range);
  const ticks = d3Scale.ticks(count);
  return ticks.map((v) => {
    const value = new TimeStamp(v).valueOf();
    return {
      value: new TimeStamp(value).valueOf(),
      offset: d3Scale(v),
      location,
      showGrid,
      height,
      type: "time",
    };
  });
};

const linearTicks = (
  size: number,
  location: OuterLocation,
  scale: Scale,
  showGrid: boolean,
  height: number,
  count: number
): TickProps[] => {
  let range = [0, size];
  const domain = [scale.pos(0), scale.pos(size)];
  if (["bottom", "right"].includes(location)) range = range.reverse();
  const tickScale = scaleLinear().domain(domain).range(range);
  const ticks = tickScale.ticks(count);
  return ticks.map((v) => {
    return {
      value: v,
      offset: tickScale(v),
      location,
      showGrid,
      height,
      type: "linear",
    };
  });
};

const calcTickCount = (size: number, pixelsPerTick: number): number => {
  const tickCount = Math.floor(size / pixelsPerTick);
  return tickCount > 0 ? tickCount : 1;
};

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
    const f = type === "time" ? timeTicks : linearTicks;
    const pxScale = scale.scale(size).reverse();
    return f(
      size,
      location,
      pxScale,
      showGrid,
      height,
      calcTickCount(size, pixelsPerTick)
    );
  }, [size, location, scale, showGrid, height, type]);
  return (
    <g transform={calcGroupTransform(location, position, size)} className="pluto-axis">
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
  const adjustedPosition = { ...position };
  if (location === "left") adjustedPosition.y += size;
  else if (location === "bottom") adjustedPosition.x += size;
  return clsx(fTranslate(adjustedPosition), fRotate(locationRotations[location]));
};

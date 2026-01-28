// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type direction, type record, xy } from "@synnaxlabs/x";
import { BaseEdge, type BaseEdgeProps } from "@xyflow/react";
import { type FC, type ReactElement } from "react";
import { z } from "zod";

import { Select } from "@/select";

export const offsetPath = (path: xy.XY[], miters: xy.XY[]): xy.XY[] =>
  path.map((point, i) => xy.translate(point, miters[i]));

interface PathProps extends Omit<BaseEdgeProps, "path" | "color" | "points"> {
  points: xy.XY[];
  color?: color.Crude;
}

const Pipe = ({ points, color: pathColor, ...rest }: PathProps): ReactElement => (
  <BaseEdge
    path={calcPath(points)}
    style={{
      stroke: color.cssString(pathColor),
    }}
    {...rest}
  />
);

const ElectricSignalPipe = ({
  points,
  color: pathColor,
  ...rest
}: PathProps): ReactElement => (
  <BaseEdge
    path={calcPath(points)}
    style={{
      stroke: color.cssString(pathColor),
      strokeDasharray: "12,4",
    }}
    {...rest}
  />
);

const SecondaryPipe = ({
  points,
  color: pathColor,
  ...rest
}: PathProps): ReactElement => (
  <BaseEdge
    path={calcPath(points)}
    style={{
      stroke: color.cssString(pathColor),
      strokeDasharray: "12,4,4",
    }}
    {...rest}
  />
);

const JackedPipe = ({ points, color: pathColor, ...rest }: PathProps): ReactElement => {
  const miters = xy.calculateMiters(points, 6);
  const abovePath = points.map((p, i) => xy.translate(p, miters[i]));
  const belowPath = points.map((p, i) => xy.translate(p, xy.scale(miters[i], -1)));
  const stroke = color.cssString(pathColor);
  const opacity = 0.7;
  return (
    <>
      <BaseEdge path={calcPath(abovePath)} style={{ stroke, opacity }} {...rest} />
      <BaseEdge path={calcPath(points)} style={{ stroke }} {...rest} />
      <BaseEdge path={calcPath(belowPath)} style={{ stroke, opacity }} {...rest} />
    </>
  );
};

const JOINT_REMOVE_THRESHOLD = 5;

const computeSymbolPositions = (points: xy.XY[], interval: number): SymbolProps[] => {
  const positions: SymbolProps[] = [];
  const segmentLengths: number[] = [];
  for (let i = 0; i < points.length - 1; i++) {
    const length = xy.distance(points[i], points[i + 1]);
    segmentLengths.push(length);
  }
  const spacing = interval;
  let currentLength = 0;
  let nextPosition = spacing;
  for (let i = 0; i < segmentLengths.length; i++) {
    const length = segmentLengths[i];
    const start = points[i];
    const end = points[i + 1];
    while (currentLength + length >= nextPosition) {
      const t = (nextPosition - currentLength) / length;
      const position = {
        x: start.x + t * (end.x - start.x),
        y: start.y + t * (end.y - start.y),
      };
      let direction: direction.Direction = "x";
      if (Math.abs(end.x - start.x) < 1) direction = "y";
      nextPosition += spacing;
      if (
        xy.distance(position, end) < JOINT_REMOVE_THRESHOLD ||
        xy.distance(position, start) < JOINT_REMOVE_THRESHOLD
      )
        continue;
      positions.push({ position, direction });
    }
    currentLength += length;
  }
  return positions;
};

interface SymbolProps {
  color?: color.Crude;
  position: xy.XY;
  direction: direction.Direction;
}

const HydraulicLSymbol = ({
  color: colorVal,
  position,
  direction,
}: SymbolProps): ReactElement => {
  const size = 10;
  const pathData = `M0,0 L0,-${size} L${size},-${size}`;
  const rotationAngle = 270;
  if (direction === "x") position.y += size / 2;
  else position.x += size / 2;
  return (
    <path
      d={pathData}
      stroke={color.cssString(colorVal)}
      fill="none"
      strokeWidth={2}
      transform={`translate(${position.x},${position.y}) rotate(${rotationAngle})`}
      strokeLinecap="round"
    />
  );
};

const ContinuousPneumaticSignalSymbol = ({
  color: colorVal,
  position,
  direction,
}: SymbolProps): ReactElement => {
  const size = 10;
  const pathData = `M0,0 L0,-${size}`;
  if (direction === "x") position.y += size / 3;
  else position.x -= size / 3;
  let pointTwo: xy.XY = xy.translateX(position, -10);
  let rotate: number = 45;
  if (direction === "y") {
    pointTwo = xy.translateY(position, -10);
    rotate += 90;
  }
  return (
    <>
      <path
        d={pathData}
        stroke={color.cssString(colorVal)}
        fill="none"
        strokeWidth={2}
        transform={`translate(${position.x},${position.y}) rotate(${rotate})`}
        strokeLinecap="round"
      />
      <path
        d={pathData}
        stroke={color.cssString(colorVal)}
        fill="none"
        strokeWidth={2}
        transform={`translate(${pointTwo.x},${pointTwo.y}) rotate(${rotate})`}
        strokeLinecap="round"
      />
    </>
  );
};

const DataLinkSymbol = ({ color: colorVal, position }: SymbolProps): ReactElement => (
  <circle
    cx={position.x}
    cy={position.y}
    r={3}
    fill="var(--pluto-gray-l0)"
    stroke={color.cssString(colorVal)}
    strokeWidth={2}
  />
);

const createSymbolLine = (C: FC<SymbolProps>) => {
  const O = ({ points, color: colorVal, ...rest }: PathProps): ReactElement => {
    const path = calcPath(points);
    const positions = computeSymbolPositions(points, 40); // Adjust the interval as needed
    return (
      <>
        <BaseEdge path={path} {...rest} style={{ stroke: color.cssString(colorVal) }} />
        {positions.map(({ position, direction }, index) => (
          <C key={index} position={position} direction={direction} color={colorVal} />
        ))}
      </>
    );
  };
  O.displayName = `SymbolLine(${C.displayName})`;
  return O;
};

const Hydraulic = createSymbolLine(HydraulicLSymbol);
const ContinuousPneumaticSignal = createSymbolLine(ContinuousPneumaticSignalSymbol);
const DataLink = createSymbolLine(DataLinkSymbol);

export const calcPath = (coords: xy.XY[]): string => {
  let path = "";
  const close = false;
  const radius = 6;
  const length = coords.length + (close ? 1 : -1);
  for (let i = 0; i < length; i++) {
    const a = coords[i % coords.length];
    const b = coords[(i + 1) % coords.length];
    const t = Math.min(radius / Math.hypot(b.x - a.x, b.y - a.y), 0.5);
    if (i > 0)
      path += `Q${a.x},${a.y} ${a.x * (1 - t) + b.x * t},${a.y * (1 - t) + b.y * t}`;
    if (!close && i === 0) path += `M${a.x},${a.y}`;
    else if (i === 0) path += `M${a.x * (1 - t) + b.x * t},${a.y * (1 - t) + b.y * t}`;
    if (!close && i === length - 1) path += `L${b.x},${b.y}`;
    else if (i < length - 1)
      path += `L${a.x * t + b.x * (1 - t)},${a.y * t + b.y * (1 - t)}`;
  }
  if (close) path += "Z";
  return path;
};

export const edgeTypeZ = z.enum([
  "pipe",
  "electric",
  "secondary",
  "jacketed",
  "hydraulic",
  "pneumatic",
  "data",
]);

export type EdgeType = z.infer<typeof edgeTypeZ>;

export const PATHS: Record<EdgeType, FC<PathProps>> = {
  pipe: Pipe,
  electric: ElectricSignalPipe,
  secondary: SecondaryPipe,
  jacketed: JackedPipe,
  hydraulic: Hydraulic,
  pneumatic: ContinuousPneumaticSignal,
  data: DataLink,
};

export const DefaultPath = Pipe;

const DATA: record.KeyedNamed<EdgeType>[] = [
  { key: "pipe", name: "Pipe" },
  { key: "electric", name: "Electric Signal" },
  { key: "secondary", name: "Secondary" },
  { key: "jacketed", name: "Jacketed" },
  { key: "hydraulic", name: "Hydraulic" },
  { key: "pneumatic", name: "Pneumatic" },
  { key: "data", name: "Data" },
];

export interface SelectEdgeTypeProps extends Omit<
  Select.StaticProps<EdgeType>,
  "data" | "resourceName"
> {}

export const SelectEdgeType = (props: SelectEdgeTypeProps): ReactElement => (
  <Select.Static {...props} data={DATA} resourceName="path type" />
);

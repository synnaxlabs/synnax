// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useId } from "react";

import { type Point, project } from "@/components/IndustrialGeometry";

type ScreenPoint = readonly [number, number];

const COMMAND: readonly Point[] = [
  [337, -42, 172],
  [337, -25, 172],
  [637, -25, 172],
  [637, 175, 172],
  [488, 175, 172],
  [488, 175, 68],
  [488, 216, 68],
  [261, 216, 68],
  [261, 216, 5],
  [118, 216, 5],
  [118, 187, 5],
];

// Offset in projected coordinates so both lanes stay visibly separated through
// horizontal runs and vertical risers. The first and last points remain shared.
const parallelLane = (
  vertices: readonly ScreenPoint[],
  separation: number,
): ScreenPoint[] => {
  const normals = vertices.slice(1).map(([x, y], i): ScreenPoint => {
    const dx = x - vertices[i][0];
    const dy = y - vertices[i][1];
    const length = Math.hypot(dx, dy);
    return [-dy / length, dx / length];
  });
  return vertices.map(([x, y], i): ScreenPoint => {
    if (i === 0 || i === vertices.length - 1) return [x, y];
    const [ax, ay] = normals[i - 1];
    const [bx, by] = normals[i];
    const miter = separation / (1 + ax * bx + ay * by);
    return [x + (ax + bx) * miter, y + (ay + by) * miter];
  });
};

const command = COMMAND.map(project);
const acknowledgment = parallelLane(command, 16).reverse();

const pathData = (vertices: readonly ScreenPoint[]): string =>
  vertices.map(([x, y], i) => `${i === 0 ? "M" : "L"}${x} ${y}`).join(" ");

const arrowSegment = (from: ScreenPoint, to: ScreenPoint): string => {
  const dx = to[0] - from[0];
  const dy = to[1] - from[1];
  const length = Math.hypot(dx, dy);
  const x = from[0] + dx * 0.56;
  const y = from[1] + dy * 0.56;
  return `M${x} ${y} l${(dx / length) * 6} ${(dy / length) * 6}`;
};

const Lane = ({
  vertices,
  arrows,
  markerId,
  className,
}: {
  vertices: readonly ScreenPoint[];
  arrows: readonly number[];
  markerId: string;
  className: string;
}): ReactElement => {
  const d = pathData(vertices);
  return (
    <g className={className}>
      <defs>
        <marker
          id={markerId}
          viewBox="0 0 8 8"
          markerWidth={8}
          markerHeight={8}
          refX={7}
          refY={4}
          orient="auto"
          markerUnits="userSpaceOnUse"
        >
          <path d="M1 1 L7 4 L1 7 Z" fill="currentColor" stroke="none" />
        </marker>
      </defs>
      <path d={d} className="scene-signal-underlay" fill="none" />
      <path d={d} className="scene-signal-path" fill="none" />
      <path d={d} className="scene-signal-packets" fill="none" />
      {arrows.map((i) => (
        <path
          key={i}
          d={arrowSegment(vertices[i], vertices[i + 1])}
          className="scene-flow-arrow"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
          markerEnd={`url(#${markerId})`}
        />
      ))}
    </g>
  );
};

export const IndustrialControlFlow = (): ReactElement => {
  const id = useId();
  return (
    <g className="scene-control-route">
      <Lane
        vertices={command}
        arrows={[1, 6]}
        markerId={`${id}-command`}
        className="scene-command"
      />
      <Lane
        vertices={acknowledgment}
        arrows={[3, 8]}
        markerId={`${id}-acknowledgment`}
        className="scene-acknowledgment"
      />
    </g>
  );
};

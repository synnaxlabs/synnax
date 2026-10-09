// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

export type Point = readonly [number, number, number];

export interface Material {
  top: string;
  left: string;
  right: string;
}

export const CONCRETE = { top: "#465364", left: "#2c3849", right: "#1b2737" };
export const LIGHT = { top: "#718299", left: "#485b73", right: "#2d4057" };
export const DARK = { top: "#29394e", left: "#1b2b40", right: "#101c2c" };
export const METAL = { top: "#667c96", left: "#3f5774", right: "#273e5c" };
export const SIGNAL = { top: "#8ab8ff", left: "#5e94ee", right: "#3774d0" };

export const project = ([x, y, z]: Point): [number, number] => [
  // Trigonometric vertices differ slightly across JS engines. Keep SVG attributes
  // identical between server rendering and browser hydration.
  Math.round((610 + (x - y) * 1.1) * 1000) / 1000,
  Math.round((150 + (x + y) * 0.47 - z) * 1000) / 1000,
];

const points = (vertices: readonly Point[]): string =>
  vertices.map((v) => project(v).join(",")).join(" ");

const path = (vertices: readonly Point[]): string =>
  vertices.map((v, i) => `${i === 0 ? "M" : "L"}${project(v).join(",")}`).join(" ");

export const Face = ({
  vertices,
  fill,
  className,
}: {
  vertices: readonly Point[];
  fill: string;
  className?: string;
}): ReactElement => (
  <polygon points={points(vertices)} fill={fill} className={className} />
);

export const Wire = ({
  vertices,
  className,
}: {
  vertices: readonly Point[];
  className?: string;
}): ReactElement => <path d={path(vertices)} className={className} fill="none" />;

export const Box = ({
  x,
  y,
  z = 0,
  width,
  depth,
  height,
  material = CONCRETE,
}: {
  x: number;
  y: number;
  z?: number;
  width: number;
  depth: number;
  height: number;
  material?: Material;
}): ReactElement => (
  <g className="scene-solid">
    <Face
      vertices={[
        [x, y + depth, z],
        [x + width, y + depth, z],
        [x + width, y + depth, z + height],
        [x, y + depth, z + height],
      ]}
      fill={material.left}
    />
    <Face
      vertices={[
        [x + width, y, z],
        [x + width, y + depth, z],
        [x + width, y + depth, z + height],
        [x + width, y, z + height],
      ]}
      fill={material.right}
    />
    <Face
      vertices={[
        [x, y, z + height],
        [x + width, y, z + height],
        [x + width, y + depth, z + height],
        [x, y + depth, z + height],
      ]}
      fill={material.top}
    />
  </g>
);

export const Cylinder = ({
  x,
  y,
  radius,
  height,
  z = 0,
  dark = false,
}: {
  x: number;
  y: number;
  radius: number;
  height: number;
  z?: number;
  dark?: boolean;
}): ReactElement => {
  const ring = (elevation: number): Point[] =>
    Array.from({ length: 32 }, (_, i) => [
      x + Math.cos((i * Math.PI) / 16) * radius,
      y + Math.sin((i * Math.PI) / 16) * radius,
      elevation,
    ]);
  const bottom = ring(z);
  const top = ring(z + height);
  return (
    <g className="scene-cylinder">
      {Array.from({ length: 16 }, (_, i) => {
        const n = (i + 28) % 32;
        const next = (n + 1) % 32;
        const shade = Math.round((dark ? 33 : 65) + 23 * Math.cos((i / 16) * Math.PI));
        return (
          <Face
            key={i}
            vertices={[bottom[n], bottom[next], top[next], top[n]]}
            fill={`rgb(${shade} ${shade + 15} ${shade + 34})`}
          />
        );
      })}
      <Face vertices={top} fill={dark ? DARK.top : LIGHT.top} />
      <Wire vertices={[...ring(z + height * 0.15), ring(z + height * 0.15)[0]]} />
      <Wire vertices={[...ring(z + height * 0.82), ring(z + height * 0.82)[0]]} />
    </g>
  );
};

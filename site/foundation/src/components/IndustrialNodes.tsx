// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import { type ReactElement } from "react";

import {
  Box,
  DARK,
  Face,
  type Point,
  project,
  Wire,
} from "@/components/IndustrialGeometry";

export const FoundationNode = ({
  x,
  y,
  label,
}: {
  x: number;
  y: number;
  label: string;
}): ReactElement => {
  const center: Point = [x + 11, y + 11, 15];
  const [sx, sy] = project(center);
  const logoPlacement = { x: -12, y: -12, width: 24, height: 24 };
  const ring = (radius: number, z: number): Point[] =>
    Array.from({ length: 6 }, (_, i) => {
      const angle = (i * Math.PI) / 3 - Math.PI / 4;
      return [
        center[0] + Math.cos(angle) * radius,
        center[1] + Math.sin(angle) * radius,
        z,
      ];
    });
  return (
    <g className="scene-foundation-node" data-node={label}>
      {[
        { radius: 23, z: 0, height: 3, material: DARK },
        {
          radius: 21,
          z: 3,
          height: 11,
          material: { top: "#30598c", left: "#25476b", right: "#19314f" },
        },
      ].map(({ radius, z, height, material }) => {
        const bottom = ring(radius, z);
        const top = ring(radius, z + height);
        return (
          <g key={radius} className="scene-solid">
            {[0, 1, 2].map((i) => (
              <Face
                key={i}
                vertices={[bottom[i], bottom[i + 1], top[i + 1], top[i]]}
                fill={i === 0 ? material.right : material.left}
              />
            ))}
            <Face vertices={top} fill={material.top} />
          </g>
        );
      })}
      <g
        transform={`translate(${sx} ${sy}) scale(${1.1 * Math.SQRT2} ${0.47 * Math.SQRT2})`}
      >
        <Logo {...logoPlacement} className="scene-node-mark" aria-hidden="true" />
      </g>
      <text x={sx} y={sy - 21} className="scene-node-id" textAnchor="middle">
        {label}
      </text>
    </g>
  );
};

export const InfrastructurePad = ({
  x,
  y,
  width,
  depth,
}: {
  x: number;
  y: number;
  width: number;
  depth: number;
}): ReactElement => (
  <g className="scene-ground">
    <Box
      x={x}
      y={y}
      z={-9}
      width={width}
      depth={depth}
      height={8}
      material={{ top: "#101720", left: "#0b1119", right: "#080e16" }}
    />
    <Wire
      vertices={[
        [x + 8, y + 8, 0],
        [x + width - 8, y + 8, 0],
        [x + width - 8, y + depth - 8, 0],
        [x + 8, y + depth - 8, 0],
        [x + 8, y + 8, 0],
      ]}
      className="scene-site-edge"
    />
  </g>
);

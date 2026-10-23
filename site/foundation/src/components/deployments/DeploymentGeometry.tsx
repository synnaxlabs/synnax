// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties, type ReactElement } from "react";

import { Box, DARK, type Point, project, Wire } from "@/components/IndustrialGeometry";
import { FoundationNode, InfrastructurePad } from "@/components/IndustrialNodes";

export const Pad = InfrastructurePad;

/** World coordinates identify the center of the node, including an optional deck height. */
export const Node = ({
  x,
  y,
  z = 0,
  label,
}: {
  x: number;
  y: number;
  z?: number;
  label: string;
}): ReactElement => (
  <g transform={`translate(0 ${-z})`}>
    <FoundationNode x={x - 11} y={y - 11} label={label} />
  </g>
);

export const Trace = ({
  vertices,
  reverse = false,
  muted = false,
  delay = 0,
}: {
  vertices: readonly Point[];
  reverse?: boolean;
  muted?: boolean;
  delay?: number;
}): ReactElement => (
  <g
    className={`deployment-trace${muted ? " deployment-trace-local" : ""}${reverse ? " deployment-trace-reverse" : ""}`}
    style={{ "--flow-delay": `${delay}s` } as CSSProperties}
  >
    <Wire vertices={vertices} className="scene-signal-underlay" />
    <Wire vertices={vertices} className="scene-signal-path" />
    <Wire vertices={vertices} className="scene-signal-packets" />
  </g>
);

export const Rack = ({
  x,
  y,
  z = 0,
  width = 28,
  depth = 28,
  height = 64,
}: {
  x: number;
  y: number;
  z?: number;
  width?: number;
  depth?: number;
  height?: number;
}): ReactElement => (
  <g>
    <Box
      x={x}
      y={y}
      z={z}
      width={width}
      depth={depth}
      height={height}
      material={DARK}
    />
    <Box
      x={x + 2}
      y={y + depth}
      z={z + 4}
      width={width - 4}
      depth={1}
      height={height - 8}
      material={{ top: "#50627a", left: "#31435a", right: "#26384f" }}
    />
    {Array.from({ length: Math.floor(height / 9) }, (_, i) => (
      <g key={i}>
        <Wire
          vertices={[
            [x + 5, y + depth + 1, z + 8 + i * 8],
            [x + width - 5, y + depth + 1, z + 8 + i * 8],
          ]}
          className="scene-rack-slot"
        />
        <Wire
          vertices={[
            [x + width - 8, y + depth + 1.5, z + 10 + i * 8],
            [x + width - 5, y + depth + 1.5, z + 10 + i * 8],
          ]}
          className="deployment-rack-indicator"
        />
      </g>
    ))}
  </g>
);

export const Label = ({
  anchor,
  x,
  y,
  title,
  detail,
  width = 220,
  side = "left",
}: {
  anchor: Point;
  x: number;
  y: number;
  title: string;
  detail?: string;
  width?: number;
  side?: "left" | "right";
}): ReactElement => {
  const [ax, ay] = project(anchor);
  const edge = side === "left" ? x + width : x;
  return (
    <g className="scene-annotation deployment-label">
      <path
        d={`M${ax} ${ay}L${edge + (side === "left" ? 18 : -18)} ${y + 14}H${edge}`}
      />
      <circle cx={ax} cy={ay} r={2} />
      <rect x={x - 6} y={y - 7} width={width + 12} height={detail ? 44 : 27} rx={3} />
      <text x={x} y={y + 9}>
        {title}
      </text>
      {detail && (
        <text x={x} y={y + 28} className="scene-annotation-detail">
          {detail}
        </text>
      )}
    </g>
  );
};

export const Datum = (): ReactElement => (
  <g className="scene-datum deployment-datum">
    {Array.from({ length: 19 }, (_, i) => (
      <Wire
        key={`x${i}`}
        vertices={[
          [i * 45 - 20, -20, -12],
          [i * 45 - 20, 540, -12],
        ]}
      />
    ))}
    {Array.from({ length: 13 }, (_, i) => (
      <Wire
        key={`y${i}`}
        vertices={[
          [-20, i * 45 - 20, -12],
          [790, i * 45 - 20, -12],
        ]}
      />
    ))}
  </g>
);

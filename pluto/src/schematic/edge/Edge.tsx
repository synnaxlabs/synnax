// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/Edge.css";

import { type schematic } from "@synnaxlabs/client";
import { color, location } from "@synnaxlabs/x";
import { useReactFlow } from "@xyflow/react";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { useKey } from "@/schematic/Context";
import { DefaultPath, PATHS } from "@/schematic/edge/paths";
import { route } from "@/schematic/edge/route";
import { useRetrieve } from "@/schematic/queries";
import { type diagram } from "@/vis/diagram/aether";

export const ConnectionLine = ({
  source,
  target,
  style,
  status,
}: diagram.ConnectionLineProps): ReactElement => {
  const connectedHandle = document.querySelector(".react-flow__handle-connecting");
  const toNodeHandle = connectedHandle?.className.match(/react-flow__handle-(\w+)/);
  if (toNodeHandle != null) {
    const res = location.outerZ.safeParse(toNodeHandle[1]);
    if (res.success) source.orientation = res.data;
  }
  const points = route({ source, target });
  return (
    <DefaultPath
      points={points}
      style={{
        ...style,
        stroke: color.cssString(
          status === "invalid" ? "var(--pluto-error-z)" : "var(--pluto-gray-l11)",
        ),
        strokeWidth: 2,
        fill: "none",
      }}
    />
  );
};

export const Edge = ({
  edgeKey,
  source,
  target,
  selected = false,
}: diagram.EdgeProps): ReactElement => {
  const key = useKey();
  const { data: doc } = useRetrieve({ key });
  const edgeProps = doc?.props?.[edgeKey] as schematic.EdgeProps | undefined;
  const {
    waypoints = [],
    color = "var(--pluto-gray-l11)",
    variant = "pipe",
  } = edgeProps ?? {};

  const flow = useReactFlow();

  const points = useMemo(
    () => route({ source, target, waypoints }),
    [source, target, waypoints],
  );

  const P = PATHS[variant] ?? PATHS.pipe;

  return (
    <>
      <P points={points} color={color} />
      {selected && (
        <g className={CSS.BE("diagram-edge", "handles")}>
          {points.slice(1, -1).map((p, i) => (
            <circle
              key={i}
              cx={p.x}
              cy={p.y}
              r={4 / flow.getZoom()}
              className={CSS.BE("diagram-edge-handle", "waypoint")}
              fill="var(--pluto-primary-z)"
              stroke="var(--pluto-gray-l0)"
              strokeWidth={1 / flow.getZoom()}
            />
          ))}
        </g>
      )}
    </>
  );
};

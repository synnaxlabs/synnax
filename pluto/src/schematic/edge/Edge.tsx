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
import { color, location, xy } from "@synnaxlabs/x";
import {
  type ConnectionLineComponentProps,
  type EdgeProps as RFEdgeProps,
  type Position,
  useReactFlow,
} from "@xyflow/react";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { DefaultPath, type EdgeType, PATHS } from "@/schematic/edge/paths";
import { route } from "@/schematic/edge/route";
import { useRetrieve } from "@/schematic/queries";

export const ConnectionLine = ({
  fromX,
  fromY,
  toX,
  toY,
  fromPosition,
  toPosition,
  connectionLineStyle,
  connectionStatus,
}: ConnectionLineComponentProps): ReactElement => {
  const connectedHandle = document.querySelector(".react-flow__handle-connecting");
  const toNodeHandle = connectedHandle?.className.match(/react-flow__handle-(\w+)/);
  if (toNodeHandle != null) {
    const res = location.outerZ.safeParse(toNodeHandle[1]);
    if (res.success) toPosition = res.data as Position;
  }
  const points = route({
    source: xy.construct(fromX, fromY),
    sourceDir: fromPosition as location.Outer,
    target: xy.construct(toX, toY),
    targetDir: toPosition as location.Outer,
  });
  return (
    <DefaultPath
      points={points}
      style={{
        ...connectionLineStyle,
        stroke: color.cssString(
          connectionStatus === "invalid"
            ? "var(--pluto-error-z)"
            : "var(--pluto-gray-l11)",
        ),
        strokeWidth: 2,
        fill: "none",
      }}
    />
  );
};

interface EdgeInternalProps extends RFEdgeProps {
  schematicKey: string;
}

export const Edge = ({
  id,
  sourcePosition: sourceOrientation,
  targetPosition: targetOrientation,
  selected = false,
  schematicKey,
  ...rest
}: EdgeInternalProps): ReactElement => {
  const { data: doc } = useRetrieve({ key: schematicKey });
  const edgeProps = doc?.props?.[id] as schematic.EdgeProps | undefined;
  const {
    waypoints = [],
    color: edgeColor = "var(--pluto-gray-l11)",
    variant = "pipe",
  } = edgeProps ?? {};

  const flow = useReactFlow();
  const sourcePos = xy.construct(rest.sourceX, rest.sourceY);
  const targetPos = xy.construct(rest.targetX, rest.targetY);

  const points = useMemo(
    () =>
      route({
        source: sourcePos,
        sourceDir: sourceOrientation as location.Outer,
        target: targetPos,
        targetDir: targetOrientation as location.Outer,
        waypoints,
      }),
    [sourcePos, targetPos, sourceOrientation, targetOrientation, waypoints],
  );

  const P = PATHS[variant as EdgeType] ?? PATHS.pipe;

  return (
    <>
      <P points={points} color={edgeColor} />
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

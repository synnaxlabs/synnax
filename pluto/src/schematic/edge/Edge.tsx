// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/Edge.css";

import { schematic } from "@synnaxlabs/client";
import { box, color, direction, location, xy } from "@synnaxlabs/x";
import { useReactFlow } from "@xyflow/react";
import {
  type DragEvent,
  Fragment,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
} from "react";

import { CSS } from "@/css";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { useKey } from "@/schematic/Context";
import { DefaultPath, PATHS } from "@/schematic/edge/paths";
import { route } from "@/schematic/edge/route";
import { useDispatch, useSelectProps } from "@/schematic/queries";
import { type Key } from "@/triggers/triggers";
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

interface CurrentlyDragging {
  /** Index of the first point of the dragged segment in the interior array. */
  segStart: number;
  /** Snapshot of the interior points at drag start. */
  initialInterior: xy.XY[];
  /** The perpendicular axis the drag moves along. */
  dragAxis: direction.Direction;
}

const segmentDir = (a: xy.XY, b: xy.XY): direction.Direction =>
  a.y === b.y ? "x" : "y";

const calcMidPoints = (points: xy.XY[]): xy.XY[] =>
  points.slice(1).map((p, i) => ({
    x: (p.x + points[i].x) / 2,
    y: (p.y + points[i].y) / 2,
  }));

export const Edge = ({
  edgeKey,
  source,
  target,
  sourceNode: _sourceNode,
  targetNode: _targetNode,
  selected = false,
}: diagram.EdgeProps): ReactElement => {
  const schematicKey = useKey();
  const edgeProps = useSelectProps({ key: schematicKey, propKey: edgeKey }) as
    | schematic.EdgeProps
    | undefined;
  const {
    waypoints: storedWaypoints = [],
    color: edgeColor = "var(--pluto-gray-l11)",
    variant = "pipe",
  } = edgeProps ?? {};

  const flow = useReactFlow();
  const { update: dispatch } = useDispatch();

  const [waypoints, setWaypoints, waypointsRef] =
    useCombinedStateAndRef<xy.XY[]>(storedWaypoints);

  const storedRef = useRef(storedWaypoints);
  if (storedWaypoints !== storedRef.current) {
    storedRef.current = storedWaypoints;
    setWaypoints(storedWaypoints);
  }

  const points = useMemo(
    () => route({ source, target, waypoints }),
    [source, target, waypoints],
  );

  const persistWaypoints = useCallback(
    (wps: xy.XY[]) => {
      dispatch({
        key: schematicKey,
        actions: schematic.setProps({
          key: edgeKey,
          props: { waypoints: wps, variant, color: edgeColor },
        }),
      });
    },
    [schematicKey, edgeKey, variant, edgeColor, dispatch],
  );

  // Refs for stable callback access - never in dependency arrays.
  const pointsRef = useSyncedRef(points);
  const flowRef = useSyncedRef(flow);
  const dragRef = useRef<CurrentlyDragging | null>(null);

  const dragStart = useCursorDrag({
    onStart: useCallback((_: xy.XY, __: Key, e: DragEvent) => {
      const segIndex = Number(e.currentTarget.id.split("-")[1]);
      const currentPoints = pointsRef.current;

      // The segment direction tells us which axis is shared and which to drag.
      const dir = segmentDir(currentPoints[segIndex], currentPoints[segIndex + 1]);
      const dragAxis = direction.swap(dir);

      // Extract all interior points (everything except source and target).
      // These become the waypoints that define the route shape.
      const interior = currentPoints.slice(1, -1).map((p) => ({ ...p }));

      // The segment at segIndex in the full points array corresponds to
      // interior indices segIndex-1 and segIndex (shifted by 1 since we
      // removed the source point).
      const interiorSegStart = segIndex - 1;

      setWaypoints(interior);
      dragRef.current = {
        segStart: interiorSegStart,
        initialInterior: interior,
        dragAxis,
      };
    }, []),
    onMove: useCallback((b: box.Box) => {
      if (dragRef.current == null) return;
      const { segStart, initialInterior, dragAxis } = dragRef.current;
      const magnitude = box.dim(b, dragAxis, true) / flowRef.current.getZoom();

      // Adjust the shared coordinate of both endpoints of the dragged segment.
      const next = initialInterior.map((p) => ({ ...p }));
      for (const idx of [segStart, segStart + 1]) {
        if (idx < 0 || idx >= next.length) continue;
        if (dragAxis === "x") next[idx] = { x: next[idx].x + magnitude, y: next[idx].y };
        else next[idx] = { x: next[idx].x, y: next[idx].y + magnitude };
      }
      setWaypoints(next);
    }, []),
    onEnd: useCallback(() => {
      persistWaypoints(waypointsRef.current);
      dragRef.current = null;
    }, [persistWaypoints]),
  });

  const P = PATHS[variant] ?? PATHS.pipe;
  const midPoints = selected ? calcMidPoints(points) : [];

  return (
    <>
      <P points={points} color={edgeColor} />
      {selected &&
        midPoints.map((p, i) => {
          if (i === 0 || i === midPoints.length - 1) return null;
          const dir = segmentDir(points[i], points[i + 1]);
          const swapped = direction.swap(dir);
          const dims = {
            [direction.dimension(dir)]: "18px",
            [direction.dimension(swapped)]: "4px",
          };
          const pos = {
            [dir]: p[dir] - 9,
            [swapped]: p[swapped] - 2,
          };
          return (
            <Fragment key={i}>
              <rect
                className={CSS.BE("diagram-edge-handle", "background")}
                fill="var(--pluto-gray-l0)"
                stroke="var(--pluto-primary-z)"
                {...dims}
                {...pos}
                rx="2px"
                ry="2px"
              />
              <foreignObject x={p.x - 9} y={p.y - 9} width="18px" height="18px">
                <div
                  id={`handle-${i}`}
                  className={CSS(
                    CSS.BE("diagram-edge-handle", "dragger"),
                    CSS.dir(dir),
                  )}
                  draggable
                  onDragStart={dragStart}
                />
              </foreignObject>
            </Fragment>
          );
        })}
    </>
  );
};


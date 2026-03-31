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
import { useCombinedStateAndRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { useKey } from "@/schematic/Context";
import { DefaultPath, PATHS } from "@/schematic/edge/paths";
import { route } from "@/schematic/edge/route";
import { useDispatch, useSelectProps } from "@/schematic/queries";
import { type Key } from "@/triggers/triggers";
import { type diagram } from "@/vis/diagram/aether";
import { selectNodeBox } from "@/vis/diagram/util";

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
  index: number;
  initialWaypoints: xy.XY[];
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
  sourceNode,
  targetNode,
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

  // Sync stored waypoints into local state when they change externally.
  const storedRef = useRef(storedWaypoints);
  if (storedWaypoints !== storedRef.current) {
    storedRef.current = storedWaypoints;
    setWaypoints(storedWaypoints);
  }

  const sourceBox = useMemo(() => {
    try {
      return selectNodeBox(flow, sourceNode);
    } catch {
      return undefined;
    }
  }, [sourceNode, source.position.x, source.position.y]);

  const targetBox = useMemo(() => {
    try {
      return selectNodeBox(flow, targetNode);
    } catch {
      return undefined;
    }
  }, [targetNode, target.position.x, target.position.y]);

  const points = useMemo(
    () => route({ source, target, waypoints, sourceBox, targetBox }),
    [source, target, waypoints, sourceBox, targetBox],
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

  const dragRef = useRef<CurrentlyDragging | null>(null);

  const dragStart = useCursorDrag({
    onStart: useCallback(
      (_: xy.XY, __: Key, e: DragEvent) => {
        const segIndex = Number(e.currentTarget.id.split("-")[1]);
        const currentWaypoints = waypointsRef.current;
        const currentPoints = route({
          source,
          target,
          waypoints: currentWaypoints,
          sourceBox,
          targetBox,
        });
        const mid = calcMidPoints(currentPoints)[segIndex];
        const insertIdx = findWaypointInsertIndex(
          currentWaypoints,
          currentPoints,
          segIndex,
        );
        const next = [...currentWaypoints];
        next.splice(insertIdx, 0, mid);
        setWaypoints(next);
        dragRef.current = { index: insertIdx, initialWaypoints: next };
      },
      [source, target, sourceBox, targetBox],
    ),
    onMove: useCallback(
      (b: box.Box) => {
        if (dragRef.current == null) return;
        const { index, initialWaypoints } = dragRef.current;
        const wp = initialWaypoints[index];
        const currentPoints = route({
          source,
          target,
          waypoints: initialWaypoints,
          sourceBox,
          targetBox,
        });

        // Find which segment this waypoint sits on to determine the drag axis.
        const pointIdx = currentPoints.findIndex((p) => xy.equals(p, wp));
        const dir =
          pointIdx > 0
            ? segmentDir(currentPoints[pointIdx - 1], currentPoints[pointIdx])
            : "x";
        const dragAxis = direction.swap(dir);

        const magnitude = box.dim(b, dragAxis, true) / flow.getZoom();

        const next = [...initialWaypoints];
        if (dragAxis === "x") next[index] = { x: wp.x + magnitude, y: wp.y };
        else next[index] = { x: wp.x, y: wp.y + magnitude };
        setWaypoints(next);
      },
      [source, target, sourceBox, targetBox],
    ),
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

const findWaypointInsertIndex = (
  waypoints: xy.XY[],
  points: xy.XY[],
  segIndex: number,
): number => {
  let wpIdx = 0;
  for (let i = 0; i <= segIndex && wpIdx < waypoints.length; i++)
    if (xy.equals(points[i], waypoints[wpIdx])) wpIdx++;
  return wpIdx;
};

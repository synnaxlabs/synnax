// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type DragEvent,
  Fragment,
  type ReactElement,
  useCallback,
  useRef,
} from "react";

import { BaseEdge, type EdgeProps as RFEdgeProps, useViewport } from "reactflow";

import { Color } from "@/color";
import { CSS } from "@/css";
import { useCombinedStateAndRef } from "@/hooks/useCombinedStateAndRef";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { type Key } from "@/triggers/triggers";
import {
  adjustToSourceOrTarget,
  handleDrag,
  calculateLineDirection,
} from "@/vis/pid/edge/edgeUtils";

import "@/vis/pid/edge/Edge.css";
import { box, direction, xy } from "@synnaxlabs/x";

interface CurrentlyDragging {
  root: xy.XY;
  index: number;
}

export interface EdgeProps extends RFEdgeProps {
  editable: boolean;
  points: xy.XY[];
  onPointsChange: (p: xy.XY[]) => void;
  color?: Color.Crude;
}

export const Edge = ({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetHandleId,
  targetPosition,
  style,
  points: propsPoints,
  onPointsChange,
  editable,
  color,
  ...props
}: EdgeProps): ReactElement => {
  const [points, setPoints, pointsRef] = useCombinedStateAndRef<xy.XY[]>(propsPoints)

  const { zoom } = useViewport();

  const adjusted = adjustToSourceOrTarget(sourceX, sourceY, targetX, targetY, points);
  if (adjusted != null) setPoints(adjusted);

  const dragRef = useRef<CurrentlyDragging | null>(null);

  const dragStart = useCursorDrag({
    onStart: useCallback((_: xy.XY, __: Key, e: DragEvent) => {
      const index = Number(e.currentTarget.id.split("-")[1]);
      dragRef.current = { root: pointsRef.current[index], index };
    }, []),
    onMove: useCallback(
      (b: box.Box) => {
        setPoints((prev) => {
          if (dragRef.current == null) return prev;
          const { root, index } = dragRef.current;
          const [nextIndex, next] = handleDrag(prev, b, root, index, zoom);
          dragRef.current.index = nextIndex;
          return next;
        });
      },
      [zoom],
    ),
    onEnd: useCallback(() => onPointsChange(pointsRef.current), [onPointsChange]),
  });

  return (
    <>
      <BaseEdge
        path={calcPath(points)}
        style={{ ...style, stroke: Color.cssString(color) }}
        {...props}
      />
      {calcMidPoints(points).map((p, i) => {
        const dir = calculateLineDirection(points[i], points[i + 1]);
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
              className={CSS.BE("pid-edge-handle", "background")}
              fill="var(--pluto-background-color)"
              stroke="var(--pluto-primary-z)"
              {...dims}
              {...pos}
              rx="2px"
              ry="2px"
            />
            <foreignObject x={p.x - 9} y={p.y - 9} width="18px" height="18px">
              <div
                id={`handle-${i}`}
                className={CSS(CSS.BE("pid-edge-handle", "dragger"), CSS.dir(dir))}
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

export const calcPath = (points: xy.XY[]): string => {
  if (points.length === 0) return "";
  const [start, ...rest] = points;
  // Generate a path string of lines between each point with a corner radius of 2px
  return `M ${start.x} ${start.y} ${rest.map((p) => `L ${p.x} ${p.y}`).join(" ")}`;
};

export const calcMidPoints = (points: xy.XY[]): xy.XY[] => {
  return points.slice(1).map((p, i) => {
    const prev = points[i];
    return xy.construct((p.x + prev.x) / 2, (p.y + prev.y) / 2);
  });
};

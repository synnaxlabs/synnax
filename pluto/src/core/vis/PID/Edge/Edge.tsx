// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DragEvent, Fragment, ReactElement, useCallback, useRef } from "react";

import { XY, CrudeXY, Box } from "@synnaxlabs/x";
import { BaseEdge, EdgeProps as RFEdgeProps, useViewport } from "reactflow";

import { Color, CrudeColor } from "@/color";
import { CSS } from "@/css";
import { useCombinedStateAndRef } from "@/hooks/useCombinedStateAndRef";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { TriggerKey } from "@/triggers/triggers";
import {
  adjustToSourceOrTarget,
  handleDrag,
  calculateLineDirection,
} from "@/core/vis/PID/Edge/edgeUtils";

import "@/core/vis/PID/Edge/Edge.css";

interface CurrentlyDragging {
  root: XY;
  index: number;
}

export interface EdgeProps extends RFEdgeProps {
  editable: boolean;
  points: CrudeXY[];
  onPointsChange: (p: CrudeXY[]) => void;
  color?: CrudeColor;
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
  const [points, setPoints, pointsRef] = useCombinedStateAndRef<XY[]>(() =>
    propsPoints.map((p) => new XY(p))
  );

  const { zoom } = useViewport();

  const adjusted = adjustToSourceOrTarget(sourceX, sourceY, targetX, targetY, points);
  if (adjusted != null) setPoints(adjusted);

  const dragRef = useRef<CurrentlyDragging | null>(null);

  const dragStart = useCursorDrag({
    onStart: useCallback((_: XY, __: TriggerKey, e: DragEvent) => {
      const index = Number(e.currentTarget.id.split("-")[1]);
      dragRef.current = { root: pointsRef.current[index], index };
    }, []),
    onMove: useCallback(
      (b: Box) => {
        setPoints((prev) => {
          if (dragRef.current == null) return prev;
          const { root, index } = dragRef.current;
          const [nextIndex, next] = handleDrag(prev, b, root, index, zoom);
          dragRef.current.index = nextIndex;
          return next;
        });
      },
      [zoom]
    ),
    onEnd: useCallback(() => onPointsChange(pointsRef.current.map((p) => p.crude)), []),
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
        const dims = {
          [dir.dimension]: "18px",
          [dir.inverse.dimension]: "4px",
        };
        const pos = {
          [dir.crude]: p[dir.crude] - 9,
          [dir.inverse.crude]: p[dir.inverse.crude] - 2,
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

export const calcPath = (points: XY[]): string => {
  if (points.length === 0) return "";
  const [start, ...rest] = points;
  // Generate a path string of lines between each point with a corner radius of 2px
  return `M ${start.x} ${start.y} ${rest.map((p) => `L ${p.x} ${p.y}`).join(" ")}`;
};

export const calcMidPoints = (points: XY[]): XY[] => {
  return points.slice(1).map((p, i) => {
    const prev = points[i];
    return new XY((p.x + prev.x) / 2, (p.y + prev.y) / 2);
  });
};

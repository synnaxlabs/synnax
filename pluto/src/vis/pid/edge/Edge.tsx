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

import { box, direction, xy } from "@synnaxlabs/x";
import {
  BaseEdge,
  type EdgeProps as RFEdgeProps,
  useViewport,
  useStore,
  useReactFlow,
  type ConnectionLineComponentProps,
} from "reactflow";

import { Color } from "@/color";
import { CSS } from "@/css";
import { useCombinedStateAndRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { Theming } from "@/theming";
import { type Key } from "@/triggers/triggers";
import {
  adjustToSourceOrTarget,
  handleDrag,
  calculateLineDirection,
  adjustToHandlePosition,
} from "@/vis/pid/edge/edgeUtils";

import { selectNode, selectNodeBox } from "../util";

import {
  type Segment,
  newConnector,
  segmentsToPoints,
  moveConnector,
} from "./connector";

import "@/vis/pid/edge/Edge.css";

interface CurrentlyDragging {
  segments: Segment[];
  index: number;
}

export interface EdgeProps extends RFEdgeProps {
  editable: boolean;
  segments: Segment[];
  onSegmentsChange: (p: xy.XY[]) => void;
  color?: Color.Crude;
  applyTransform?: boolean;
}

export const CustomConnectionLine = ({
  fromX,
  fromY,
  toX,
  toY,
  fromPosition,
  toPosition,
  fromNode,
  connectionLineStyle,
}: ConnectionLineComponentProps): ReactElement => {
  console.log(toPosition, fromNode);
  const t = Theming.use();
  return (
    <Edge
      sourceX={fromX}
      sourceY={fromY}
      targetX={toX}
      targetY={toY}
      sourcePosition={fromPosition}
      targetPosition={toPosition}
      color={t.colors.gray.l9}
      editable={false}
      segments={[]}
      onSegmentsChange={() => {}}
      id="custom-connection"
      source={fromNode?.id ?? ""}
      target={fromNode?.id ?? ""}
      style={{ strokeWidth: 2 }}
      applyTransform={false}
    />
  );
};

export const Edge = ({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  source,
  target,
  sourcePosition,
  targetHandleId,
  targetPosition,
  style,
  segments: propsSegments,
  onSegmentsChange: onPointsChange,
  editable,
  color,
  applyTransform = true,
  ...props
}: EdgeProps): ReactElement => {
  const flow = useReactFlow();
  const [segments, setSegments, segRef] = useCombinedStateAndRef<Segment[]>(
    propsSegments?.length > 0
      ? propsSegments
      : newConnector({
          sourcePos: xy.construct(sourceX, sourceY),
          targetPos: xy.construct(targetX, targetY),
          sourceOrientation: sourcePosition,
          targetOrientation: targetPosition,
          sourceBox: selectNodeBox(flow, source),
          targetBox: selectNodeBox(flow, target),
        }),
  );

  // const adjusted = adjustToSourceOrTarget(sourceX, sourceY, targetX, targetY, points);
  // if (adjusted != null) setPoints(adjusted);

  const dragRef = useRef<CurrentlyDragging | null>(null);

  const dragStart = useCursorDrag({
    onStart: useCallback((_: xy.XY, __: Key, e: DragEvent) => {
      dragRef.current = {
        index: Number(e.currentTarget.id.split("-")[1]),
        segments: [...segRef.current],
      };
      console.log(dragRef.current);
    }, []),
    onMove: useCallback((b: box.Box) => {
      if (dragRef.current == null) return;
      const next = moveConnector({
        segments: dragRef.current.segments,
        index: dragRef.current.index,
        magnitude: box.dim(
          b,
          direction.swap(dragRef.current.segments[dragRef.current.index].direction),
          true,
        ),
      });
      setSegments(next);
    }, []),
    onEnd: useCallback(() => onPointsChange(pointsRef.current), [onPointsChange]),
  });

  // points = adjustToHandlePosition(points, sourcePosition, targetPosition);

  const points = segmentsToPoints(
    { x: sourceX, y: sourceY },
    segments,
    flow.getZoom(),
    applyTransform,
  );

  return (
    <>
      <BaseEdge
        path={calcPath(points)}
        style={{ ...style, stroke: Color.cssString(color) }}
        interactionWidth={6}
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

export const calcPath = (coords: xy.XY[]): string => {
  let path = "";
  const close = false;
  const radius = 6;
  const length = coords.length + (close ? 1 : -1);
  for (let i = 0; i < length; i++) {
    const a = coords[i % coords.length];
    const b = coords[(i + 1) % coords.length];
    const t = Math.min(radius / Math.hypot(b.x - a.x, b.y - a.y), 0.5);

    if (i > 0)
      path += `Q${a.x},${a.y} ${a.x * (1 - t) + b.x * t},${a.y * (1 - t) + b.y * t}`;

    if (!close && i == 0) path += `M${a.x},${a.y}`;
    else if (i == 0) path += `M${a.x * (1 - t) + b.x * t},${a.y * (1 - t) + b.y * t}`;

    if (!close && i == length - 1) path += `L${b.x},${b.y}`;
    else if (i < length - 1)
      path += `L${a.x * t + b.x * (1 - t)},${a.y * t + b.y * (1 - t)}`;
  }
  if (close) path += "Z";
  return path;
};

export const calcMidPoints = (points: xy.XY[]): xy.XY[] => {
  return points.slice(1).map((p, i) => {
    const prev = points[i];
    return xy.construct((p.x + prev.x) / 2, (p.y + prev.y) / 2);
  });
};

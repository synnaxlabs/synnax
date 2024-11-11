// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/diagram/edge/Edge.css";

import { box, direction, location, xy } from "@synnaxlabs/x";
import {
  BaseEdge,
  type ConnectionLineComponentProps,
  type EdgeProps as RFEdgeProps,
  type Position,
  useReactFlow,
} from "@xyflow/react";
import {
  type DragEvent,
  Fragment,
  type ReactElement,
  useCallback,
  useRef,
} from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { useCombinedStateAndRef, useDebouncedCallback } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { type Key } from "@/triggers/triggers";
import { connector } from "@/vis/diagram/edge/connector";
import { selectNodeBox } from "@/vis/diagram/util";

interface CurrentlyDragging {
  segments: connector.Segment[];
  index: number;
}

export interface EdgeProps extends RFEdgeProps {
  segments: connector.Segment[];
  onSegmentsChange: (segments: connector.Segment[]) => void;
  color?: Color.Crude;
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
  connectionStatus,
}: ConnectionLineComponentProps): ReactElement => {
  // select an element with 'react-flow__handle-connecting' class
  const connectedHandle = document.querySelector(".react-flow__handle-connecting");
  const toNodeHandle = connectedHandle?.className.match(/react-flow__handle-(\w+)/);
  if (toNodeHandle != null) {
    const res = location.outer.safeParse(toNodeHandle[1]);
    if (res.success) toPosition = res.data as Position;
  }
  const flow = useReactFlow();
  const conn = connector.buildNew({
    sourcePos: xy.construct(fromX, fromY),
    targetPos: xy.construct(toX, toY),
    sourceOrientation: fromPosition,
    targetOrientation: toPosition,
    sourceBox: selectNodeBox(flow, fromNode?.id ?? ""),
    targetBox: selectNodeBox(flow, fromNode?.id ?? ""),
  });
  const points = connector.segmentsToPoints(
    xy.construct(fromX, fromY),
    conn,
    flow.getZoom(),
    false,
  );

  return (
    <BaseEdge
      path={calcPath(points)}
      style={{
        ...connectionLineStyle,
        stroke: Color.cssString(
          connectionStatus === "invalid"
            ? "var(--pluto-error-z)"
            : "var(--pluto-gray-l9)",
        ),
        strokeWidth: 2,
        fill: "none",
      }}
    />
  );
};

export const Edge = ({
  id,
  source,
  target,
  sourcePosition: sourceOrientation,
  targetHandleId,
  targetPosition: targetOrientation,
  style,
  segments: propsSegments = [],
  onSegmentsChange,
  color = "var(--pluto-gray-l9)",
  selected = false,
  ...props
}: EdgeProps): ReactElement => {
  const sourcePos = xy.construct(props.sourceX, props.sourceY);
  const sourcePosRef = useRef(sourcePos);
  const sourcePosEq = xy.equals(sourcePos, sourcePosRef.current);

  const targetPos = xy.construct(props.targetX, props.targetY);
  const targetPosRef = useRef(targetPos);
  const targetPosEq = xy.equals(targetPos, targetPosRef.current);

  const flow = useReactFlow();
  const [segments, setSegments, segRef] = useCombinedStateAndRef<connector.Segment[]>(
    () =>
      propsSegments.length > 0
        ? propsSegments
        : connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation,
            targetOrientation,
            sourceBox: selectNodeBox(flow, source),
            targetBox: selectNodeBox(flow, target),
          }),
  );

  const targetOrientationRef = useRef(targetOrientation);
  const sourceOrientationRef = useRef(sourceOrientation);

  const debouncedOnSegmentsChange = useDebouncedCallback(onSegmentsChange, 100, [
    onSegmentsChange,
  ]);

  if (!sourcePosEq || !targetPosEq) {
    let next: connector.Segment[] = segments;
    const sourceDelta = xy.translation(sourcePosRef.current, sourcePos);
    const targetDelta = xy.translation(targetPos, targetPosRef.current);
    if (xy.equals(sourceDelta, xy.scale(targetDelta, -1), 0.001)) {
      sourcePosRef.current = sourcePos;
      targetPosRef.current = targetPos;
    } else {
      if (!sourcePosEq) {
        next = connector.moveSourceNode({ delta: sourceDelta, segments: next });
        if (sourceOrientationRef.current !== sourceOrientation) {
          sourceOrientationRef.current = sourceOrientation;
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation,
            targetOrientation,
            sourceBox: selectNodeBox(flow, source),
            targetBox: selectNodeBox(flow, target),
          });
        }
        if (!connector.checkIntegrity({ sourcePos, targetPos, next, prev: segments })) {
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation,
            targetOrientation,
            sourceBox: selectNodeBox(flow, source),
            targetBox: selectNodeBox(flow, target),
          });
        }
        sourcePosRef.current = sourcePos;
      } else if (!targetPosEq) {
        next = connector.moveTargetNode({ delta: targetDelta, segments: next });
        if (targetOrientationRef.current !== targetOrientation) {
          targetOrientationRef.current = targetOrientation;
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation,
            targetOrientation,
            sourceBox: selectNodeBox(flow, source),
            targetBox: selectNodeBox(flow, target),
          });
        }
        if (!connector.checkIntegrity({ sourcePos, targetPos, next, prev: segments })) {
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation,
            targetOrientation,
            sourceBox: selectNodeBox(flow, source),
            targetBox: selectNodeBox(flow, target),
          });
        }
        targetPosRef.current = targetPos;
      }
      debouncedOnSegmentsChange(next);
      setSegments(next);
    }
  }

  const dragRef = useRef<CurrentlyDragging | null>(null);

  const dragStart = useCursorDrag({
    onStart: useCallback((_: xy.XY, __: Key, e: DragEvent) => {
      dragRef.current = {
        index: Number(e.currentTarget.id.split("-")[1]),
        segments: [...segRef.current],
      };
    }, []),
    onMove: useCallback((b: box.Box) => {
      if (dragRef.current == null) return;
      const next = connector.dragSegment({
        segments: dragRef.current.segments,
        index: dragRef.current.index,
        magnitude:
          box.dim(
            b,
            direction.swap(dragRef.current.segments[dragRef.current.index].direction),
            true,
          ) / flow.getZoom(),
      });
      setSegments(next);
    }, []),
    onEnd: useCallback(() => onSegmentsChange(segRef.current), [onSegmentsChange]),
  });

  const points = connector.segmentsToPoints(sourcePos, segments, flow.getZoom(), true);

  return (
    <>
      <BaseEdge
        path={calcPath(points)}
        style={{ ...style, stroke: Color.cssString(color) }}
        {...props}
      />
      {selected &&
        calcMidPoints(points).map((p, i) => {
          const dir = segments[i].direction;
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
    if (!close && i === 0) path += `M${a.x},${a.y}`;
    else if (i === 0) path += `M${a.x * (1 - t) + b.x * t},${a.y * (1 - t) + b.y * t}`;
    if (!close && i === length - 1) path += `L${b.x},${b.y}`;
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

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/Edge.css";

import { box, color, direction, location, type record, xy } from "@synnaxlabs/x";
import {
  type ConnectionLineComponentProps,
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

import { CSS } from "@/css";
import { useCombinedStateAndRef, useDebouncedCallback } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { connector } from "@/schematic/edge/connector";
import { DefaultPath, type EdgeType, PATHS } from "@/schematic/edge/paths";
import { type Key } from "@/triggers/triggers";
import { type Diagram } from "@/vis/diagram";
import { selectNodeBox } from "@/vis/diagram/util";

interface CurrentlyDragging {
  segments: connector.Segment[];
  index: number;
}

export interface EdgeData extends record.Unknown {
  segments: connector.Segment[];
  variant?: EdgeType;
  color?: color.Crude;
}

export const ConnectionLine = ({
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
    const res = location.outerZ.safeParse(toNodeHandle[1]);
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

export const Edge = ({
  id,
  source,
  target,
  sourcePosition: sourceOrientation,
  targetHandleId,
  targetPosition: targetOrientation,
  style,
  data,
  selected = false,
  onDataChange,
  ...rest
}: Diagram.EdgeProps<EdgeData>): ReactElement => {
  const {
    segments: propsSegments = [],
    color = "var(--pluto-gray-l11)",
    variant = "pipe",
  } = data ?? {};
  const sourcePos = xy.construct(rest.sourceX, rest.sourceY);
  const sourcePosRef = useRef(sourcePos);
  const sourcePosEq = xy.equals(sourcePos, sourcePosRef.current);

  const targetPos = xy.construct(rest.targetX, rest.targetY);
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

  const handleSegmentsChange = useCallback(
    (segments: connector.Segment[]) => {
      onDataChange({ ...data, segments });
    },
    [data, onDataChange],
  );

  const debouncedOnSegmentsChange = useDebouncedCallback(handleSegmentsChange, 100, [
    handleSegmentsChange,
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
        if (!connector.checkIntegrity({ sourcePos, targetPos, next, prev: segments }))
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation,
            targetOrientation,
            sourceBox: selectNodeBox(flow, source),
            targetBox: selectNodeBox(flow, target),
          });
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
        if (!connector.checkIntegrity({ sourcePos, targetPos, next, prev: segments }))
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation,
            targetOrientation,
            sourceBox: selectNodeBox(flow, source),
            targetBox: selectNodeBox(flow, target),
          });
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
    onEnd: useCallback(
      () => handleSegmentsChange(segRef.current),
      [handleSegmentsChange],
    ),
  });

  const points = connector.segmentsToPoints(sourcePos, segments, flow.getZoom(), true);

  const P = PATHS[variant];

  return (
    <>
      <P points={points} color={color} />
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

export const calcMidPoints = (points: xy.XY[]): xy.XY[] =>
  points.slice(1).map((p, i) => {
    const prev = points[i];
    return xy.construct((p.x + prev.x) / 2, (p.y + prev.y) / 2);
  });

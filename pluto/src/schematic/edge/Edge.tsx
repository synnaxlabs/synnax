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
  useRef,
} from "react";

import { CSS } from "@/css";
import { useCombinedStateAndRef, useDebouncedCallback } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { useKey } from "@/schematic/Context";
import { connector } from "@/schematic/edge/connector";
import { DefaultPath, PATHS } from "@/schematic/edge/paths";
import { useDispatch, useSelectProps } from "@/schematic/queries";
import { type Key } from "@/triggers/triggers";
import { type diagram } from "@/vis/diagram/aether";
import { selectNodeBox } from "@/vis/diagram/util";

interface CurrentlyDragging {
  segments: connector.Segment[];
  index: number;
}

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
    if (res.success) target.orientation = res.data;
  }
  const flow = useReactFlow();
  const conn = connector.buildNew({
    sourcePos: source.position,
    targetPos: target.position,
    sourceOrientation: source.orientation,
    targetOrientation: target.orientation,
    sourceBox: box.ZERO,
    targetBox: box.ZERO,
  });
  const points = connector.segmentsToPoints(
    source.position,
    conn,
    flow.getZoom(),
    false,
  );
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
  sourceNode,
  targetNode,
  selected = false,
}: diagram.EdgeProps): ReactElement => {
  const schematicKey = useKey();
  const edgeProps = useSelectProps({ key: schematicKey, propKey: edgeKey }) as
    | schematic.EdgeProps
    | undefined;
  const {
    segments: propsSegments = [],
    color: edgeColor = "var(--pluto-gray-l11)",
    variant = "pipe",
  } = (edgeProps ?? {}) as schematic.EdgeProps;

  const sourcePos = source.position;
  const sourcePosRef = useRef(sourcePos);
  const sourcePosEq = xy.equals(sourcePos, sourcePosRef.current);

  const targetPos = target.position;
  const targetPosRef = useRef(targetPos);
  const targetPosEq = xy.equals(targetPos, targetPosRef.current);

  const flow = useReactFlow();
  const { update: dispatch } = useDispatch();

  const [segments, setSegments, segRef] = useCombinedStateAndRef<connector.Segment[]>(
    () =>
      propsSegments.length > 0
        ? propsSegments
        : connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation: source.orientation,
            targetOrientation: target.orientation,
            sourceBox: selectNodeBox(flow, sourceNode),
            targetBox: selectNodeBox(flow, targetNode),
          }),
  );

  const propsRef = useRef(propsSegments);
  if (propsSegments !== propsRef.current) {
    propsRef.current = propsSegments;
    if (propsSegments.length > 0) setSegments(propsSegments);
  }

  const targetOrientationRef = useRef(target.orientation);
  const sourceOrientationRef = useRef(source.orientation);

  const persistSegments = useCallback(
    (segs: connector.Segment[]) => {
      console.log("persistSegments", { segs, variant, edgeColor });
      dispatch({
        key: schematicKey,
        actions: schematic.setProps({
          key: edgeKey,
          props: { segments: segs, variant, color: edgeColor },
        }),
      });
    },
    [schematicKey, edgeKey, variant, edgeColor, dispatch],
  );

  const debouncedPersist = useDebouncedCallback(persistSegments, 100, [
    persistSegments,
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
        if (sourceOrientationRef.current !== source.orientation) {
          sourceOrientationRef.current = source.orientation;
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation: source.orientation,
            targetOrientation: target.orientation,
            sourceBox: selectNodeBox(flow, sourceNode),
            targetBox: selectNodeBox(flow, targetNode),
          });
        }
        if (!connector.checkIntegrity({ sourcePos, targetPos, next, prev: segments }))
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation: source.orientation,
            targetOrientation: target.orientation,
            sourceBox: selectNodeBox(flow, sourceNode),
            targetBox: selectNodeBox(flow, targetNode),
          });
        sourcePosRef.current = sourcePos;
      } else if (!targetPosEq) {
        next = connector.moveTargetNode({ delta: targetDelta, segments: next });
        if (targetOrientationRef.current !== target.orientation) {
          targetOrientationRef.current = target.orientation;
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation: source.orientation,
            targetOrientation: target.orientation,
            sourceBox: selectNodeBox(flow, sourceNode),
            targetBox: selectNodeBox(flow, targetNode),
          });
        }
        if (!connector.checkIntegrity({ sourcePos, targetPos, next, prev: segments }))
          next = connector.buildNew({
            sourcePos,
            targetPos,
            sourceOrientation: source.orientation,
            targetOrientation: target.orientation,
            sourceBox: selectNodeBox(flow, sourceNode),
            targetBox: selectNodeBox(flow, targetNode),
          });
        targetPosRef.current = targetPos;
      }
      debouncedPersist(next);
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
    onEnd: useCallback(() => persistSegments(segRef.current), [persistSegments]),
  });

  const points = connector.segmentsToPoints(sourcePos, segments, flow.getZoom(), true);

  const P = PATHS[variant] ?? PATHS.pipe;

  return (
    <>
      <P points={points} color={edgeColor} />
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

const calcMidPoints = (points: xy.XY[]): xy.XY[] =>
  points.slice(1).map((p, i) => {
    const prev = points[i];
    return xy.construct((p.x + prev.x) / 2, (p.y + prev.y) / 2);
  });

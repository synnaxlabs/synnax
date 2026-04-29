// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/Edge.css";

import { box, color, direction, location, xy } from "@synnaxlabs/x";
import { useReactFlow } from "@xyflow/react";
import {
  type DragEvent,
  Fragment,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { connector } from "@/schematic/edge/connector";
import { DefaultPath, type EdgeType, PATHS } from "@/schematic/edge/paths";
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

export interface EdgeProps extends diagram.EdgeProps {
  segments?: connector.Segment[];
  variant?: EdgeType;
  color?: color.Crude;
  onSegmentsChange: (segments: connector.Segment[]) => void;
}

export const Edge = ({
  edgeKey,
  source,
  target,
  sourceNode,
  targetNode,
  selected = false,
  segments: middleSegments = [],
  variant = "pipe",
  color: edgeColor = "var(--pluto-gray-l11)",
  onSegmentsChange,
}: EdgeProps): ReactElement | null => {
  const flow = useReactFlow();

  const visualSegments = useMemo(() => {
    if (middleSegments.length === 0)
      return connector.buildNew({
        sourcePos: source.position,
        targetPos: target.position,
        sourceOrientation: source.orientation,
        targetOrientation: target.orientation,
        sourceBox: selectNodeBox(flow, sourceNode),
        targetBox: selectNodeBox(flow, targetNode),
      });
    return connector.stitchEdge({
      sourceOrientation: source.orientation,
      targetOrientation: target.orientation,
      sourcePos: source.position,
      targetPos: target.position,
      middleSegments,
    });
  }, [
    source.position.x,
    source.position.y,
    target.position.x,
    target.position.y,
    source.orientation,
    target.orientation,
    middleSegments,
  ]);

  const persistMiddle = useCallback(
    (segs: connector.Segment[]) => {
      const middle = connector.extractMiddle(
        segs,
        source.orientation,
        target.orientation,
      );
      onSegmentsChange(middle);
    },
    [source.orientation, target.orientation, onSegmentsChange],
  );

  const [dragOverride, setDragOverride] = useState<connector.Segment[] | null>(null);
  const dragOverrideRef = useRef(dragOverride);
  dragOverrideRef.current = dragOverride;

  const segments = dragOverride ?? visualSegments;
  const dragRef = useRef<CurrentlyDragging | null>(null);

  const dragStart = useCursorDrag({
    onStart: useCallback(
      (_: xy.XY, __: Key, e: DragEvent) => {
        dragRef.current = {
          index: Number(e.currentTarget.id.split("-")[1]),
          segments: [...segments],
        };
      },
      [segments],
    ),
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
      setDragOverride(next);
    }, []),
    onEnd: useCallback(() => {
      if (dragOverrideRef.current != null) {
        persistMiddle(dragOverrideRef.current);
        setDragOverride(null);
      }
    }, [persistMiddle]),
  });

  const points = connector.segmentsToPoints(
    source.position,
    segments,
    flow.getZoom(),
    true,
  );

  if (segments.length === 0) return null;

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

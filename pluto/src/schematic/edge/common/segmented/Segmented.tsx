// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/common/segmented/Segmented.css";

import { type schematic } from "@synnaxlabs/client";
import { box, direction, xy } from "@synnaxlabs/x";
import { useReactFlow } from "@xyflow/react";
import {
  type DragEvent,
  type FC,
  Fragment,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { type Base } from "@/schematic/edge/common/base";
import {
  type Config,
  createConfigZ,
  createDefaultConfig,
} from "@/schematic/edge/common/segmented/config";
import {
  createConnector,
  dragSegment,
  extractMiddle,
  type Segment,
  segmentsToPoints,
  stitchEdge,
} from "@/schematic/edge/common/segmented/connector";
import { Form } from "@/schematic/edge/common/segmented/Form";
import { type Edge, type Spec } from "@/schematic/edge/spec";
import { type Key } from "@/triggers/triggers";
import { selectNodeBox } from "@/vis/diagram/util";

interface CurrentlyDragging {
  segments: Segment[];
  index: number;
}

export interface PathProps extends Omit<Base.BaseProps, "path" | "points"> {
  points: xy.XY[];
}

const create = <V extends schematic.EdgeConfigType>(
  Path: FC<PathProps>,
): Edge<Config<V>> => {
  const E: Edge<Config<V>> = ({
    source,
    target,
    sourceNode,
    targetNode,
    selected = false,
    config: { segments: middleSegments, color: edgeColor },
    onChange,
  }): ReactElement | null => {
    const flow = useReactFlow();
    const visualSegments = useMemo(() => {
      if (middleSegments.length === 0)
        return createConnector({
          sourcePos: source.position,
          targetPos: target.position,
          sourceOrientation: source.orientation,
          targetOrientation: target.orientation,
          sourceBox: selectNodeBox(flow, sourceNode),
          targetBox: selectNodeBox(flow, targetNode),
        });
      return stitchEdge({
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
      (segs: Segment[]) => {
        onChange({
          segments: extractMiddle(segs, source.orientation, target.orientation),
        });
      },
      [source.orientation, target.orientation, onChange],
    );

    const [dragOverride, setDragOverride] = useState<Segment[] | null>(null);
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
        const next = dragSegment({
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

    const points = segmentsToPoints(source.position, segments, flow.getZoom(), true);

    if (segments.length === 0) return null;

    return (
      <>
        <Path points={points} color={edgeColor} />
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
  E.displayName = `Edge(${Path.displayName ?? Path.name})`;
  return E;
};

const calcMidPoints = (points: xy.XY[]): xy.XY[] =>
  points.slice(1).map((p, i) => {
    const prev = points[i];
    return xy.construct((p.x + prev.x) / 2, (p.y + prev.y) / 2);
  });

export const createSpec = <V extends schematic.EdgeConfigType>(
  variant: V,
  name: string,
  path: FC<PathProps>,
): Spec<V, Config<V>> => ({
  key: variant,
  name,
  configZ: createConfigZ(variant),
  Edge: create<V>(path),
  Form,
  defaultConfig: () => createDefaultConfig(variant),
});

export { type Config, createConfigZ, createDefaultConfig };

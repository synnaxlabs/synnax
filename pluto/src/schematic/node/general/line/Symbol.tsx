// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { useStoreApi } from "@xyflow/react";
import { type ReactElement, useRef } from "react";

import { CSS } from "@/css";
import { Cursor } from "@/cursor";
import { Grid } from "@/schematic/node/common/grid";
import { type Config } from "@/schematic/node/general/line/config";
import { Line } from "@/schematic/node/general/line/Primitive";
import { type NodeProps } from "@/schematic/node/spec";

// Absolute endpoints when the drag began, and which one is moving.
interface Drag {
  index: number;
  points: xy.XY[];
}

// Snaps to horizontal or vertical, whichever is closer.
const snap = (p: xy.XY, to: xy.XY): xy.XY =>
  Math.abs(p.x - to.x) >= Math.abs(p.y - to.y) ? { ...p, y: to.y } : { ...p, x: to.x };

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  draggable,
  position = xy.ZERO,
  config: { color, start, end, strokeWidth },
}: NodeProps<Config>): ReactElement => {
  const store = useStoreApi();
  const dragRef = useRef<Drag | null>(null);
  const points = [start, end];

  const startDrag = Cursor.useDrag({
    onStart: (_, __, el) => {
      dragRef.current = {
        index: Number(el.dataset.index),
        points: points.map((p) => xy.translate(position, p)),
      };
    },
    onMove: (b, _, e) => {
      const drag = dragRef.current;
      if (drag == null) return;
      const zoom = store.getState().transform[2];
      const { signedWidth, signedHeight } = box.signedDims(b);
      const fixed = drag.points[1 - drag.index];
      let moved = xy.translate(drag.points[drag.index], {
        x: signedWidth / zoom,
        y: signedHeight / zoom,
      });
      if (e.shiftKey) moved = snap(moved, fixed);
      const next = drag.points.with(drag.index, moved);
      const origin = xy.round(box.topLeft(box.construct(next[0], next[1])));
      // Position is not config, so it takes react-flow's change pipeline like a drag.
      store
        .getState()
        .triggerNodeChanges([{ type: "position", id: nodeKey, position: origin }]);
      const rel = next.map((p) => xy.round(xy.sub(p, origin)));
      onConfigChange({ start: rel[0], end: rel[1] });
    },
  });

  return (
    <>
      <Line
        className={Grid.DRAG_HANDLE_CLASS}
        color={color}
        start={start}
        end={end}
        strokeWidth={strokeWidth}
      />
      {selected &&
        draggable !== false &&
        points.map((p, i) => (
          <div
            key={i}
            data-index={i}
            className={CSS.cls("nodrag", Cursor.DRAG_CLASS, CSS.BE("line", "end"))}
            style={{ left: p.x, top: p.y }}
            onPointerDown={startDrag}
          />
        ))}
    </>
  );
};

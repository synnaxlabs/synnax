// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type direction, location, xy } from "@synnaxlabs/x";

import { type NodeLayout } from "@/vis/diagram/util";

/** Aligns nodes to a specific edge (left/right/top/bottom). */
export const alignNodesToLocation = (
  layouts: NodeLayout[],
  loc: location.Outer,
): NodeLayout[] => {
  if (layouts.length === 0) return layouts;

  const alignDir = location.direction(loc);
  const isMin = loc === "left" || loc === "top";

  const target = isMin
    ? Math.min(...layouts.map((l) => box.loc(l.box, loc)))
    : Math.max(...layouts.map((l) => box.loc(l.box, loc)));

  layouts.forEach((layout) => {
    const offset = target - box.loc(layout.box, loc);
    const newPos = xy.translate(
      box.topLeft(layout.box),
      alignDir === "x" ? { x: offset, y: 0 } : { x: 0, y: offset },
    );
    layout.box = box.construct(newPos, box.dims(layout.box));
  });
  return layouts;
};

/** Aligns nodes by their handles along a direction (x or y). */
export const alignNodesAlongDirection = (
  layouts: NodeLayout[],
  dir: direction.Direction = "x",
): NodeLayout[] => {
  if (layouts.length === 0) return layouts;

  const loc = location.construct(dir);
  const oppositeLoc = location.swap(loc);
  // Sort layouts by position, lowest to highest
  layouts.sort((a, b) => box.loc(a.box, loc) - box.loc(b.box, loc));

  // Pre-compute handles in direction for each layout
  const handlesInDir = layouts.map((layout) =>
    layout.handles.filter(
      (h) => h.orientation === loc || h.orientation === oppositeLoc,
    ),
  );

  layouts.forEach((layout, i) => {
    if (i === 0) return;
    const prev = layouts[i - 1];

    const prevHandlesInDir = handlesInDir[i - 1];
    const currentHandlesInDir = handlesInDir[i];

    if (prevHandlesInDir.length === 0 || currentHandlesInDir.length === 0) {
      const prevCenter = box.center(prev.box);
      const currentCenter = box.center(layout.box);
      const dist = xy.set(xy.translation(currentCenter, prevCenter), dir, 0);
      const newPos = xy.translate(box.topLeft(layout.box), dist);
      layout.box = box.construct(newPos, box.dims(layout.box));
      return;
    }

    // Align current node's handle with previous node's handle
    const prevHandle =
      prevHandlesInDir.find((h) => h.orientation === oppositeLoc) ??
      prevHandlesInDir[prevHandlesInDir.length - 1];
    const currentHandle =
      currentHandlesInDir.find((h) => h.orientation === loc) ?? currentHandlesInDir[0];

    const dist = xy.set(
      xy.translation(currentHandle.absolutePosition, prevHandle.absolutePosition),
      dir,
      0,
    );
    const newPos = xy.translate(box.topLeft(layout.box), dist);
    layout.box = box.construct(newPos, box.dims(layout.box));
  });
  return layouts;
};

export const distributeNodes = (
  layouts: NodeLayout[],
  dir: direction.Direction,
): NodeLayout[] => {
  if (layouts.length <= 2) return layouts;

  const loc = location.construct(dir);
  const oppositeLoc = location.swap(loc);

  const sorted = [...layouts].sort((a, b) => box.loc(a.box, loc) - box.loc(b.box, loc));

  const first = sorted[0];
  const last = sorted[sorted.length - 1];
  const middleNodes = sorted.slice(1, -1);

  const totalSpace = box.loc(last.box, loc) - box.loc(first.box, oppositeLoc);

  const totalMiddleSize = middleNodes.reduce(
    (sum, node) => sum + box.dim(node.box, dir),
    0,
  );

  const numGaps = sorted.length - 1;
  const gapSize = (totalSpace - totalMiddleSize) / numGaps;

  // Distribute middle nodes evenly between first and last (even if gap is negative)
  let current = box.loc(first.box, oppositeLoc) + gapSize;
  middleNodes.forEach((node) => {
    const pos = xy.construct(
      dir === "x" ? current : box.loc(node.box, "left"),
      dir === "y" ? current : box.loc(node.box, "top"),
    );
    node.box = box.construct(pos, box.dims(node.box));
    current += box.dim(node.box, dir) + gapSize;
  });

  return layouts;
};

export const rotateNodesAroundCenter = (
  layouts: NodeLayout[],
  dir: direction.Angular,
): NodeLayout[] => {
  if (layouts.length === 0) return [];

  const minX = Math.min(...layouts.map((l) => box.left(l.box)));
  const maxX = Math.max(...layouts.map((l) => box.right(l.box)));
  const minY = Math.min(...layouts.map((l) => box.top(l.box)));
  const maxY = Math.max(...layouts.map((l) => box.bottom(l.box)));

  const center = {
    x: (minX + maxX) / 2,
    y: (minY + maxY) / 2,
  };

  layouts.forEach((layout) => {
    const nodeCenter = box.center(layout.box);
    const newCenter = xy.rotate(nodeCenter, center, dir);
    const dims = box.dims(layout.box);
    const newPos = {
      x: newCenter.x - dims.width / 2,
      y: newCenter.y - dims.height / 2,
    };
    layout.box = box.construct(newPos, dims);
  });

  return layouts;
};

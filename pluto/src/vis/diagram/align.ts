// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, direction, location, xy } from "@synnaxlabs/x";

import { type NodeLayout } from "@/vis/diagram/util";

export const alignNodes = (
  layouts: NodeLayout[],
  dir: direction.Crude = "x",
): NodeLayout[] => {
  if (layouts.length === 0) return [];

  if (dir === "left" || dir === "right" || dir === "top" || dir === "bottom") {
    const loc = location.construct(dir);
    const alignDir = location.direction(loc);
    const isMin = dir === "left" || dir === "top";

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
  }

  const alignDir = direction.construct(dir);
  const loc = location.construct(alignDir);
  const oppositeLoc = location.swap(loc);
  layouts.sort((a, b) => box.loc(a.box, loc) - box.loc(b.box, loc));
  layouts.forEach((layout, i) => {
    if (i === 0) return;
    const prev = layouts[i - 1];

    const prevHandlesInDir = prev.handles.filter(
      (h) => h.orientation === loc || h.orientation === oppositeLoc,
    );
    const currentHandlesInDir = layout.handles.filter(
      (h) => h.orientation === loc || h.orientation === oppositeLoc,
    );

    if (prevHandlesInDir.length === 0 || currentHandlesInDir.length === 0) {
      const prevCenter = box.center(prev.box);
      const currentCenter = box.center(layout.box);
      const dist = xy.set(xy.translation(currentCenter, prevCenter), alignDir, 0);
      const newPos = xy.translate(box.topLeft(layout.box), dist);
      layout.box = box.construct(newPos, box.dims(layout.box));
      return;
    }

    const prevHandle =
      prevHandlesInDir.find((h) => h.orientation === oppositeLoc) ??
      prevHandlesInDir[prevHandlesInDir.length - 1];
    const currentHandle =
      currentHandlesInDir.find((h) => h.orientation === loc) ?? currentHandlesInDir[0];

    const dist = xy.set(
      xy.translation(currentHandle.absolutePosition, prevHandle.absolutePosition),
      alignDir,
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
  const oppositeDir = direction.swap(dir);

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
  const rawGapSize = (totalSpace - totalMiddleSize) / numGaps;

  if (rawGapSize < 0) {
    const remaining = sorted
      .slice(1)
      .sort(
        (a, b) =>
          box.loc(a.box, location.construct(oppositeDir)) -
          box.loc(b.box, location.construct(oppositeDir)),
      );
    const reordered = [first, ...remaining];
    let current = box.loc(first.box, loc);

    reordered.forEach((node) => {
      const pos = xy.construct(
        dir === "x" ? current : box.loc(node.box, location.construct("x")),
        dir === "y" ? current : box.loc(node.box, location.construct("y")),
      );
      node.box = box.construct(pos, box.dims(node.box));
      current += box.dim(node.box, dir);
    });
  } else {
    let current = box.loc(first.box, oppositeLoc) + rawGapSize;
    middleNodes.forEach((node) => {
      const pos = xy.construct(
        dir === "x" ? current : box.loc(node.box, location.construct("x")),
        dir === "y" ? current : box.loc(node.box, location.construct("y")),
      );
      node.box = box.construct(pos, box.dims(node.box));
      current += box.dim(node.box, dir) + rawGapSize;
    });
  }

  return layouts;
};

export const rotateNodes = (
  _layouts: NodeLayout[],
  _dir: direction.Angular,
): NodeLayout[] => _layouts;

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
    const newCenter = direction.rotate(nodeCenter, center, dir);
    const dims = box.dims(layout.box);
    const newPos = {
      x: newCenter.x - dims.width / 2,
      y: newCenter.y - dims.height / 2,
    };
    layout.box = box.construct(newPos, dims);
  });

  return layouts;
};

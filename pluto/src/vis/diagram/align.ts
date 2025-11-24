// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type direction, location, xy } from "@synnaxlabs/x";

import { type NodeLayout } from "@/vis/diagram/util";

export const alignNodes = (
  layouts: NodeLayout[],
  dir: direction.Direction | "left" | "right" | "top" | "bottom" = "x",
): NodeLayout[] => {
  if (layouts.length === 0) return [];

  // Handle edge alignment (left/right)
  if (dir === "left") {
    const minLeft = Math.min(...layouts.map((l) => box.left(l.box)));
    layouts.forEach((layout) => {
      const currentLeft = box.left(layout.box);
      const offset = minLeft - currentLeft;
      const newPos = xy.translate(box.topLeft(layout.box), { x: offset, y: 0 });
      layout.box = box.construct(newPos, box.dims(layout.box));
    });
    return layouts;
  }

  if (dir === "right") {
    const maxRight = Math.max(...layouts.map((l) => box.right(l.box)));
    layouts.forEach((layout) => {
      const currentRight = box.right(layout.box);
      const offset = maxRight - currentRight;
      const newPos = xy.translate(box.topLeft(layout.box), { x: offset, y: 0 });
      layout.box = box.construct(newPos, box.dims(layout.box));
    });
    return layouts;
  }

  if (dir === "top") {
    const minTop = Math.min(...layouts.map((l) => box.top(l.box)));
    layouts.forEach((layout) => {
      const currentTop = box.top(layout.box);
      const offset = minTop - currentTop;
      const newPos = xy.translate(box.topLeft(layout.box), { x: 0, y: offset });
      layout.box = box.construct(newPos, box.dims(layout.box));
    });
    return layouts;
  }

  if (dir === "bottom") {
    const maxBottom = Math.max(...layouts.map((l) => box.bottom(l.box)));
    layouts.forEach((layout) => {
      const currentBottom = box.bottom(layout.box);
      const offset = maxBottom - currentBottom;
      const newPos = xy.translate(box.topLeft(layout.box), { x: 0, y: offset });
      layout.box = box.construct(newPos, box.dims(layout.box));
    });
    return layouts;
  }

  // Handle center alignment by handles (existing behavior)
  const loc = location.construct(dir);
  const oppositeLoc = location.swap(loc);
  // sort the layouts in order of their x position, lowest to highest
  layouts.sort((a, b) => box.loc(a.box, loc) - box.loc(b.box, loc));
  // grab the first node
  layouts.forEach((layout, i) => {
    if (i === 0) return;
    const prev = layouts[i - 1];
    const prevRightHandle = prev.handles.find((h) => h.orientation === oppositeLoc);
    const leftHandle = layout.handles.find((h) => h.orientation === loc);

    // If handles don't exist in the primary direction, fall back to perpendicular handles
    if (prevRightHandle == null || leftHandle == null) {
      // For horizontal alignment (x), use midpoint of top/bottom handles if left/right don't exist
      // For vertical alignment (y), use midpoint of left/right handles if top/bottom don't exist
      const perpendicularDir = dir === "x" ? "y" : "x";
      const perpendicularLoc1 = location.construct(
        perpendicularDir === "x" ? "left" : "top",
      );
      const perpendicularLoc2 = location.construct(
        perpendicularDir === "x" ? "right" : "bottom",
      );

      const prevHandles = [
        prev.handles.find((h) => h.orientation === perpendicularLoc1),
        prev.handles.find((h) => h.orientation === perpendicularLoc2),
      ].filter((h) => h != null);

      const currentHandles = [
        layout.handles.find((h) => h.orientation === perpendicularLoc1),
        layout.handles.find((h) => h.orientation === perpendicularLoc2),
      ].filter((h) => h != null);

      if (prevHandles.length === 0 || currentHandles.length === 0) return;

      // Calculate midpoint of perpendicular handles
      const prevMidpoint = prevHandles.reduce(
        (sum, h) => xy.translate(sum, h.absolutePosition),
        { x: 0, y: 0 },
      );
      const prevAvg = xy.scale(prevMidpoint, 1 / prevHandles.length);

      const currentMidpoint = currentHandles.reduce(
        (sum, h) => xy.translate(sum, h.absolutePosition),
        { x: 0, y: 0 },
      );
      const currentAvg = xy.scale(currentMidpoint, 1 / currentHandles.length);

      const dist = xy.set(xy.translation(currentAvg, prevAvg), dir, 0);
      const newPos = xy.translate(box.topLeft(layout.box), dist);
      layout.box = box.construct(newPos, box.dims(layout.box));
      return;
    }

    // We want to align the left handle of the current node with the right handle of the previous node
    const prevHandleAbsPos = prevRightHandle.absolutePosition;
    const leftHandleAbsPos = leftHandle.absolutePosition;
    const dist = xy.set(xy.translation(leftHandleAbsPos, prevHandleAbsPos), dir, 0);
    const newPos = xy.translate(box.topLeft(layout.box), dist);
    layout.box = box.construct(newPos, box.dims(layout.box));
  });
  return layouts;
};

export const distributeNodes = (
  layouts: NodeLayout[],
  dir: "horizontal" | "vertical",
): NodeLayout[] => {
  if (layouts.length <= 2) return layouts;

  if (dir === "horizontal") {
    // Sort by left edge position
    const sorted = [...layouts].sort((a, b) => box.left(a.box) - box.left(b.box));

    // Keep first node fixed as anchor
    const first = sorted[0];
    const last = sorted[sorted.length - 1];

    // Calculate total available space
    const totalSpace = box.left(last.box) - box.right(first.box);

    // Calculate total width of middle nodes
    const middleNodes = sorted.slice(1, -1);
    const totalMiddleWidth = middleNodes.reduce(
      (sum, node) => sum + box.width(node.box),
      0,
    );

    // Calculate gap size
    const numGaps = sorted.length - 1;
    const rawGapSize = (totalSpace - totalMiddleWidth) / numGaps;

    if (rawGapSize < 0) {
      // Not enough space - anchor leftmost node and stack with 0 gap
      // Keep leftmost node first, then sort remaining nodes by vertical position
      const remaining = sorted.slice(1).sort((a, b) => box.top(a.box) - box.top(b.box));
      const reordered = [first, ...remaining];
      let currentX = box.left(first.box);

      reordered.forEach((node) => {
        const newPos = xy.construct(currentX, box.top(node.box));
        node.box = box.construct(newPos, box.dims(node.box));
        currentX += box.width(node.box);
      });
    } else {
      // Enough space - keep first and last fixed, distribute middle nodes
      let currentX = box.right(first.box) + rawGapSize;
      middleNodes.forEach((node) => {
        const newPos = xy.construct(currentX, box.top(node.box));
        node.box = box.construct(newPos, box.dims(node.box));
        currentX += box.width(node.box) + rawGapSize;
      });
    }

    return layouts;
  }

  // Vertical distribution
  // Sort by top edge position
  const sorted = [...layouts].sort((a, b) => box.top(a.box) - box.top(b.box));

  // Keep first node fixed as anchor
  const first = sorted[0];
  const last = sorted[sorted.length - 1];

  // Calculate total available space
  const totalSpace = box.top(last.box) - box.bottom(first.box);

  // Calculate total height of middle nodes
  const middleNodes = sorted.slice(1, -1);
  const totalMiddleHeight = middleNodes.reduce(
    (sum, node) => sum + box.height(node.box),
    0,
  );

  // Calculate gap size
  const numGaps = sorted.length - 1;
  const rawGapSize = (totalSpace - totalMiddleHeight) / numGaps;

  if (rawGapSize < 0) {
    // Not enough space - anchor only the first node and stack with 0 gap
    let currentY = box.bottom(first.box);
    sorted.slice(1).forEach((node) => {
      const newPos = xy.construct(box.left(node.box), currentY);
      node.box = box.construct(newPos, box.dims(node.box));
      currentY += box.height(node.box);
    });
  } else {
    // Enough space - keep first and last fixed, distribute middle nodes
    let currentY = box.bottom(first.box) + rawGapSize;
    middleNodes.forEach((node) => {
      const newPos = xy.construct(box.left(node.box), currentY);
      node.box = box.construct(newPos, box.dims(node.box));
      currentY += box.height(node.box) + rawGapSize;
    });
  }

  return layouts;
};

export const rotateNodes = (
  layouts: NodeLayout[],
  dir: "clockwise" | "counterclockwise",
): NodeLayout[] => {
  if (layouts.length === 0) return [];

  layouts.forEach((layout) => {
    try {
      const nodeElement = document.querySelector(`[data-id="${layout.key}"]`);
      if (!nodeElement) return;

      const rotateButton = nodeElement.querySelector(".pluto-grid__rotate");
      if (!rotateButton) return; // Skip nodes without rotate capability

      const clickCount = dir === "clockwise" ? 1 : 3;

      for (let i = 0; i < clickCount; i++) {
        setTimeout(() => {
          (rotateButton as HTMLElement).click();
        }, i * 5); // 5ms delay between clicks
      }
    } catch (e) {
      console.warn(`Failed to rotate node ${layout.key}:`, e);
    }
  });

  return layouts;
};

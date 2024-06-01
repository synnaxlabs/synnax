// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type direction, location,xy } from "@synnaxlabs/x";

import { type NodeLayout } from "@/vis/diagram/util";

export const alignNodes = (
  layouts: NodeLayout[],
  dir: direction.Direction = "x",
): NodeLayout[] => {
  if (layouts.length === 0) return [];
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
    if (prevRightHandle == null || leftHandle == null) return;
    // We want to align the left handle of the current node with the right handle of the previous node
    const prevHandleAbsPos = prevRightHandle.absolutePosition;
    const leftHandleAbsPos = leftHandle.absolutePosition;
    const dist = xy.set(xy.translation(leftHandleAbsPos, prevHandleAbsPos), dir, 0);
    const newPos = xy.translate(box.topLeft(layout.box), dist);
    layout.box = box.construct(newPos, box.dims(layout.box));
  });
  return layouts;
};

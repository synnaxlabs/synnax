// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type DragEvent, useCallback } from "react";

import { Haul } from "@/haul";
import { createTabDropHaulItem } from "@/mosaic/haul";

export interface UseDragTabReturn {
  /**
   * startDrag begins hauling the tab with the given key. Call from a draggable
   * tab element's onDragStart handler. Anything passed as data reaches the drop
   * handler untouched, including a handler in another window.
   */
  startDrag: (e: DragEvent<HTMLElement>, tabKey: string, data?: record.Unknown) => void;
  /** onDragEnd resolves the drag. Spread onto the same draggable tab element. */
  onDragEnd: (e: DragEvent<HTMLElement>) => void;
}

/**
 * useDragTab wires tab elements up as mosaic drag sources. The returned handlers
 * can be shared by every tab in a strip: pass the tab's key to startDrag from its
 * onDragStart, and attach onDragEnd alongside it.
 */
export const useDragTab = (): UseDragTabReturn => {
  const { startDrag, onDragEnd } = Haul.useDrag({ type: "Mosaic" });
  const handleStart = useCallback(
    (e: DragEvent<HTMLElement>, tabKey: string, data?: record.Unknown) =>
      startDrag([createTabDropHaulItem(tabKey, e.currentTarget.id, data)]),
    [startDrag],
  );
  return { startDrag: handleStart, onDragEnd };
};

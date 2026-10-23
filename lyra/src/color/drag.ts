// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, xy } from "@synnaxlabs/x";
import { type RefObject, useCallback } from "react";

import { Cursor } from "@/cursor";
import { useSyncedRef } from "@/hooks";

const UNIT = bounds.construct(0, 1);

/**
 * Calls `onChange` with the pointer's position inside the element, normalized to 0-1 on
 * each axis and clamped to the element, from the press through every move.
 */
export const useNormalizedDrag = (
  ref: RefObject<HTMLElement | null>,
  onChange: (position: xy.XY) => void,
): void => {
  // A drag keeps the handlers it started with, so they read the latest onChange.
  const onChangeRef = useSyncedRef(onChange);
  const handlePoint = useCallback((point: xy.XY) => {
    if (ref.current == null) return;
    const b = box.construct(ref.current);
    onChangeRef.current({
      x: bounds.clamp(UNIT, (point.x - box.left(b)) / box.width(b)),
      y: bounds.clamp(UNIT, (point.y - box.top(b)) / box.height(b)),
    });
  }, []);
  Cursor.useVirtualDrag({
    ref,
    onStart: handlePoint,
    onMove: useCallback(
      (_: box.Box, __: unknown, e: PointerEvent) => handlePoint(xy.construct(e)),
      [handlePoint],
    ),
  });
};

/**
 * @returns the step an arrow key moves a 0-1 value, or null for any other key. Shift
 * moves ten times as far.
 */
export const keyStep = (e: { key: string; shiftKey: boolean }): number | null => {
  const size = e.shiftKey ? 0.1 : 0.01;
  switch (e.key) {
    case "ArrowRight":
    case "ArrowUp":
      return size;
    case "ArrowLeft":
    case "ArrowDown":
      return -size;
    default:
      return null;
  }
};

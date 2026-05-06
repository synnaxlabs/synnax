// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type DragEvent, useCallback } from "react";

import { Triggers } from "@/triggers";

export interface UseCursorDragProps {
  onStart?: (loc: xy.XY, mouseKey: Triggers.Key, e: DragEvent) => void;
  onMove?: (box: box.Box, mouseKey: Triggers.Key, e: MouseEvent) => void;
  onEnd?: (box: box.Box, mouseKey: Triggers.Key, e: MouseEvent) => void;
}

export type UseCursorDragStart = (e: DragEvent) => void;

export const useCursorDrag = ({
  onMove,
  onStart,
  onEnd,
}: UseCursorDragProps): UseCursorDragStart =>
  useCallback(
    (e) => {
      e.preventDefault();
      const startLoc = xy.construct(e);
      const mouseKey = Triggers.mouseKey(e.button);
      onStart?.(startLoc, mouseKey, e);
      const handleMove = (e: MouseEvent): void => {
        if (e.buttons === 0) return handleUp(e);
        const next = xy.construct(e);
        onMove?.(box.construct(startLoc, next), mouseKey, e);
      };
      window.addEventListener("mousemove", handleMove);
      const handleUp = (e: xy.Crude): void => {
        window.removeEventListener("mousemove", handleMove);
        const next = xy.construct(e);
        onEnd?.(box.construct(startLoc, next), mouseKey, e as MouseEvent);
      };
      window.addEventListener("mouseup", handleUp, { once: true });
    },
    [onMove, onStart, onEnd],
  );

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DragEvent, useCallback } from "react";

import { Box, ClientXYT, XY } from "@synnaxlabs/x";

import { Triggers, TriggerKey } from "@/core/triggers";

export interface UseCursorDragProps {
  onStart?: (loc: XY, mouseKey: TriggerKey, e: DragEvent) => void;
  onMove?: (box: Box, mouseKey: TriggerKey, e: MouseEvent) => void;
  onEnd?: (box: Box, mouseKey: TriggerKey, e: MouseEvent) => void;
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
      const startLoc = new XY(e);
      const mouseKey = Triggers.mouseKey(e.button);
      onStart?.(startLoc, mouseKey, e);
      const handleMove = (e: MouseEvent): void => {
        if (e.buttons === 0) return handleUp(e);
        const next = new XY(e);
        onMove?.(new Box(startLoc, next), mouseKey, e);
      };
      window.addEventListener("mousemove", handleMove);
      const handleUp = (e: ClientXYT): void => {
        window.removeEventListener("mousemove", handleMove);
        const next = new XY(e);
        onEnd?.(new Box(startLoc, next), mouseKey, e as MouseEvent);
      };
      window.addEventListener("mouseup", handleUp, { once: true });
    },
    [onMove, onStart, onEnd]
  );

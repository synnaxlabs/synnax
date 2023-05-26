// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { Box, ClientXY, toXY, XY } from "@synnaxlabs/x";

import { Triggers, Key } from "@/core/triggers";

export interface UseCursorDragProps {
  onStart?: (loc: XY, mouseKey: Key) => void;
  onMove?: (box: Box, mouseKey: Key) => void;
  onEnd?: (box: Box, mouseKey: Key) => void;
}

export type UseCursorDragStart = (
  e: ClientXY & { button: number; preventDefault: () => void }
) => void;

export const useCursorDrag = ({
  onMove,
  onStart,
  onEnd,
}: UseCursorDragProps): UseCursorDragStart =>
  useCallback(
    (e) => {
      e.preventDefault();
      const startLoc = toXY(e);
      const mouseKey = Triggers.mouseKey(e.button);
      onStart?.(startLoc, mouseKey);
      const handleMove = (e: ClientXY & { buttons: number }): void => {
        if (e.buttons === 0) return handleUp(e);
        const next = toXY(e);
        onMove?.(new Box(startLoc, next), mouseKey);
      };
      window.addEventListener("mousemove", handleMove);
      const handleUp = (e: ClientXY): void => {
        window.removeEventListener("mousemove", handleMove);
        onEnd?.(new Box(startLoc, toXY(e)), mouseKey);
      };
      window.addEventListener("mouseup", handleUp, { once: true });
    },
    [onMove, onStart, onEnd]
  );

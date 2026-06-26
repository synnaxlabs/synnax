// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cursor/drag.css";

import { box, xy } from "@synnaxlabs/x";
import { type PointerEvent as ReactPointerEvent, useCallback } from "react";

import { CSS } from "@/css";
import { Triggers } from "@/triggers";

/**
 * Class applying the `touch-action`/`user-select` rules a cursor-drag element needs. Add
 * it to whatever element receives the handler returned by {@link useDrag}.
 */
export const DRAG_CLASS = CSS.B("cursor-drag");

export interface UseDragProps {
  /**
   * Called once when the pointer has moved past the activation threshold, marking the
   * true start of a drag. Receives the press location, the mouse button, and the element
   * the gesture started on.
   */
  onStart?: (loc: xy.XY, mouseKey: Triggers.Key, el: HTMLElement) => void;
  /** Called on every pointer move during a drag with the box from start to current. */
  onMove?: (box: box.Box, mouseKey: Triggers.Key, e: PointerEvent) => void;
  /** Called once when the drag ends (pointer up or cancel). Not called for a click. */
  onEnd?: (box: box.Box, mouseKey: Triggers.Key, e: PointerEvent) => void;
}

export type UseDragStart = (e: ReactPointerEvent) => void;

/** Distance in pixels the pointer must travel before a press is treated as a drag. */
const DRAG_THRESHOLD = 4;

/**
 * Returns a handler to spread onto an element's `onPointerDown` that drives a pointer
 * drag. The gesture activates only after the pointer moves past {@link DRAG_THRESHOLD},
 * so a stationary press stays a click. Once active, the pointer is captured to the
 * element so moves and the terminating up/cancel are delivered even outside its bounds.
 *
 * Add {@link DRAG_CLASS} to the element so touch gestures and text selection don't
 * pre-empt the drag.
 */
export const useDrag = ({ onMove, onStart, onEnd }: UseDragProps): UseDragStart =>
  useCallback(
    (e) => {
      if (e.button !== 0 || e.isPrimary === false) return;
      const el = e.currentTarget as HTMLElement;
      const { pointerId } = e;
      const start = xy.construct(e);
      const mouseKey = Triggers.eventKey(e);
      let started = false;

      const handleMove = (me: PointerEvent): void => {
        if (me.pointerId !== pointerId) return;
        const next = xy.construct(me);
        if (!started) {
          if (Math.hypot(next.x - start.x, next.y - start.y) < DRAG_THRESHOLD) return;
          started = true;
          el.setPointerCapture(pointerId);
          onStart?.(start, mouseKey, el);
        }
        onMove?.(box.construct(start, next), mouseKey, me);
      };

      const handleEnd = (ee: PointerEvent): void => {
        if (ee.pointerId !== pointerId) return;
        window.removeEventListener("pointermove", handleMove);
        window.removeEventListener("pointerup", handleEnd);
        window.removeEventListener("pointercancel", handleEnd);
        if (!started) return;
        if (el.hasPointerCapture(pointerId)) el.releasePointerCapture(pointerId);
        onEnd?.(box.construct(start, xy.construct(ee)), mouseKey, ee);
      };

      window.addEventListener("pointermove", handleMove);
      window.addEventListener("pointerup", handleEnd);
      window.addEventListener("pointercancel", handleEnd);
    },
    [onMove, onStart, onEnd],
  );

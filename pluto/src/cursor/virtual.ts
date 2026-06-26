// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type RefObject, useEffect } from "react";

import { type UseDragProps } from "@/cursor/drag";
import { useStateRef } from "@/hooks/ref";
import { Triggers } from "@/triggers";

export interface UseVirtualDragProps extends UseDragProps {
  ref: RefObject<HTMLElement | null>;
}

interface RefState {
  start: xy.XY;
  mouseKey: Triggers.Key;
}

const INITIAL_STATE: RefState = {
  start: xy.ZERO,
  mouseKey: "MouseLeft",
};

/**
 * A variant of {@link useDrag} that attaches its own `pointerdown` listener to a ref'd
 * element, for cases where the gesture initiator cannot receive an `onPointerDown` prop
 * directly. Activates immediately on press (no movement threshold).
 */
export const useVirtualDrag = ({
  ref,
  onMove,
  onStart,
  onEnd,
}: UseVirtualDragProps): void => {
  const [stateRef, setRef] = useStateRef<RefState>(INITIAL_STATE);
  useEffect(() => {
    if (ref.current == null) return;
    const { current: el } = ref;

    const handleMove = (e: PointerEvent): void => {
      const next = xy.construct(e);
      const { mouseKey, start } = stateRef.current;
      onMove?.(box.construct(start, next), mouseKey, e);
    };

    const handleDown = (e: PointerEvent): void => {
      el.setPointerCapture(e.pointerId);
      el.onpointermove = handleMove;
      const start = xy.construct(e);
      const mouseKey = Triggers.eventKey(e);
      setRef({ start, mouseKey });
      onStart?.(start, mouseKey, el);
      el.addEventListener("pointerup", handleUp, { once: true });
    };
    el.addEventListener("pointerdown", handleDown);

    const handleUp = (e: PointerEvent): void => {
      el.onpointermove = null;
      el.releasePointerCapture(e.pointerId);
      const { start, mouseKey } = stateRef.current;
      onEnd?.(box.construct(start, xy.construct(e)), mouseKey, e);
    };

    return () => el.removeEventListener("pointerdown", handleDown);
  }, [onMove, onStart, onEnd]);
};

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";

import { XY, ZERO_XY, toXY, Box } from "@synnaxlabs/x";

import { UseVirtualCursorDragProps } from "./types";

import { useStateRef } from "@/hooks/useStateRef";
import { Triggers, Key } from "@/triggers";

interface RefState {
  start: XY;
  mouseKey: Key;
}

const INITIAL_STATE: RefState = {
  start: ZERO_XY,
  mouseKey: "MouseLeft",
};

export const useVirtualCursorDragWebKit = ({
  ref,
  onMove,
  onStart,
  onEnd,
}: UseVirtualCursorDragProps): void => {
  const [stateRef, setRef] = useStateRef<RefState>(INITIAL_STATE);
  useEffect(() => {
    if (ref.current == null) return;
    const { current: el } = ref;

    const handleMove = (e: MouseEvent): void => {
      const next = toXY(e);
      const { mouseKey, start } = stateRef.current;
      onMove?.(new Box(start, next), mouseKey);
    };

    const handleDown = (e: PointerEvent): void => {
      el.setPointerCapture(e.pointerId);
      el.onpointermove = handleMove;
      const start = toXY(e);
      const mouseKey = Triggers.eventKey(e);
      setRef({ start, mouseKey });
      onStart?.(start, mouseKey);
      el.addEventListener("pointerup", handleUp, { once: true });
    };
    el.addEventListener("pointerdown", handleDown);

    const handleUp = (e: PointerEvent): void => {
      el.onpointermove = null;
      el.releasePointerCapture(e.pointerId);
      const { start, mouseKey } = stateRef.current;
      onEnd?.(new Box(start, toXY(e)), mouseKey);
    };

    return () => el.removeEventListener("pointerdown", handleDown);
  }, [onMove, onStart, onEnd]);
};

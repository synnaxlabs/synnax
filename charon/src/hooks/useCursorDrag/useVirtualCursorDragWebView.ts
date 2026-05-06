// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type DragEvent, useEffect } from "react";

import { useStateRef } from "@/hooks";
import { type UseVirtualCursorDragProps } from "@/hooks/useCursorDrag/types";
import { Triggers } from "@/triggers";

interface RefState {
  start: xy.XY;
  current: xy.XY;
  mouseKey: Triggers.Key;
  cursor: HTMLElement | null;
}

const INITIAL_STATE: RefState = {
  start: xy.ZERO,
  current: xy.ZERO,
  mouseKey: "MouseLeft",
  cursor: null,
};

export const useVirtualCursorDragWebView = ({
  ref,
  onMove,
  onStart,
  onEnd,
}: UseVirtualCursorDragProps): void => {
  const [stateRef, setRef] = useStateRef<RefState>(INITIAL_STATE);
  useEffect(() => {
    if (ref.current == null) return;
    const { current: el } = ref;

    const onMouseDown = (e: MouseEvent): void => {
      if (document.pointerLockElement != null) return;
      const start = xy.construct(e);
      const mouseKey = Triggers.eventKey(e);
      onStart?.(start, mouseKey, e as unknown as DragEvent);

      // push a cursor onto the document
      document.body.style.cursor = "none";
      const cursor = document.createElement("div");
      cursor.id = "cursor";
      cursor.style.position = "fixed";
      cursor.style.width = "10px";
      cursor.style.height = "10px";
      cursor.style.borderRadius = "50%";
      cursor.style.backgroundColor = "var(--pluto-text-color)";
      cursor.style.pointerEvents = "none";
      cursor.style.top = `${e.clientY}px`;
      cursor.style.left = `${e.clientX}px`;
      document.body.appendChild(cursor);

      setRef({ start, current: start, mouseKey, cursor });

      void el.requestPointerLock();
    };

    el.addEventListener("mousedown", onMouseDown);

    document.addEventListener("pointerlockchange", () => {
      if (document.pointerLockElement !== el) return;
      const { start, mouseKey } = stateRef.current;

      const handleMove = (e: MouseEvent): void => {
        setRef((prev) => {
          if (prev.cursor == null) return prev;

          const current = {
            x: prev.current.x + e.movementX,
            y: prev.current.y + e.movementY,
          };

          if (current.y > document.body.clientHeight) current.y = 0;
          if (current.y < 0) current.y = window.innerHeight;
          if (current.x > document.body.clientWidth) current.x = 0;
          if (current.x < 0) current.x = window.innerWidth;

          prev.cursor.style.top = `${current.y}px`;
          prev.cursor.style.left = `${current.x}px`;
          onMove?.(box.construct(start, current), mouseKey, e);
          return { ...prev, current: xy.construct(current) };
        });
      };
      document.addEventListener("mousemove", handleMove);

      const handleUp = (e: MouseEvent): void => {
        document.removeEventListener("mousemove", handleMove);
        document.getElementById("cursor")?.remove();
        document.body.style.cursor = "";
        document.exitPointerLock();
        onEnd?.(box.construct(stateRef.current.start, xy.construct(e)), mouseKey, e);
      };
      document.addEventListener("mouseup", handleUp, { once: true });
    });

    return () => el.removeEventListener("mousedown", onMouseDown);
  }, [onMove, onStart, onEnd]);
};

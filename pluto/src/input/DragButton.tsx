// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useMemo, useRef } from "react";

import { xy, type direction, box } from "@synnaxlabs/x";
import { GrDrag } from "react-icons/gr";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Cursor } from "@/cursor";
import { useVirtualCursorDrag } from "@/hooks/useCursorDrag";
import { type Control } from "@/input/types";

import "@/input/DragButton.css";

export interface DragButtonExtensionProps {
  direction?: direction.Crude;
  dragDirection?: direction.Crude;
  dragScale?: xy.Crude | number;
  dragThreshold?: xy.Crude | number;
  resetValue?: number;
}

export interface DragButtonProps
  extends Omit<
      Button.IconProps,
      "direction" | "onChange" | "onDragStart" | "children" | "value"
    >,
    Control<number>,
    DragButtonExtensionProps {}

export const DragButton = ({
  direction,
  className,
  dragScale = { x: 10, y: 1 },
  dragThreshold = 15,
  dragDirection,
  onChange,
  value,
  size,
  resetValue,
  ...props
}: DragButtonProps): ReactElement => {
  const vRef = useRef({
    dragging: false,
    curr: value,
    prev: value,
  });
  const elRef = useRef<HTMLButtonElement>(null);

  if (!vRef.current.dragging) vRef.current.prev = value;

  const normalDragScale = useMemo(() => {
    const scale = xy.construct(dragScale);
    if (dragDirection === "x") return xy.construct(scale.x, 0);
    if (dragDirection === "y") return xy.construct(0, scale.y);
    return scale;
  }, [dragScale, dragDirection]);
  const normalDragThreshold = useMemo(
    () => (dragThreshold != null ? xy.construct(dragThreshold) : null),
    [dragThreshold],
  );

  useVirtualCursorDrag({
    ref: elRef,
    onMove: useCallback(
      (b: box.Box) => {
        if (elRef.current == null) return;
        let value = vRef.current.prev;
        vRef.current.dragging = true;
        const { x, y } =
          normalDragThreshold ?? xy.construct(box.dims(box.construct(elRef.current)));
        if (box.width(b) > x && box.width(b) > box.height(b)) {
          const offset = box.signedWidth(b) < 0 ? x : -x;
          value += (box.signedWidth(b) + offset) * normalDragScale.x;
          Cursor.setGlobalStyle("ew-resize");
        }
        if (box.height(b) > y && box.height(b) > box.width(b)) {
          const offset = box.signedHeight(b) < 0 ? y : -y;
          value += (box.signedHeight(b) + offset) * normalDragScale.y;
          Cursor.setGlobalStyle("ns-resize");
        }
        vRef.current.curr = value;
        onChange(value);
      },
      [onChange, normalDragScale, normalDragThreshold],
    ),
    onEnd: useCallback(() => {
      vRef.current.prev = vRef.current.curr;
      vRef.current.dragging = false;
      Cursor.clearGlobalStyle();
    }, []),
  });

  const handleDoubleClick = useCallback(() => {
    onChange(resetValue ?? vRef.current.prev);
  }, [onChange, resetValue]);

  return (
    <Button.Icon
      ref={elRef}
      variant="outlined"
      className={CSS(CSS.BE("input", "drag-btn"), CSS.dir(direction), className)}
      onDoubleClick={handleDoubleClick}
      onClick={(e) => e.preventDefault()}
      {...props}
    >
      <GrDrag />
    </Button.Icon>
  );
};

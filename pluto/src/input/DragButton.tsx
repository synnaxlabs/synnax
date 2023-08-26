// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useMemo, useRef } from "react";

import { Box, CrudeDirection, LooseXYT, XY } from "@synnaxlabs/x";
import { GrDrag } from "react-icons/gr";

import { Button } from "@/button";
import { CSS } from "@/css";
import { useVirtualCursorDrag } from "@/hooks/useCursorDrag";
import { Control } from "@/input/types";

import "@/input/DragButton.css";

export interface DragButtonExtensionProps {
  direction?: CrudeDirection;
  dragDirection?: CrudeDirection;
  dragScale?: LooseXYT | number;
  dragThreshold?: LooseXYT | number;
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
    const scale = new XY(dragScale);
    if (dragDirection === "x") return new XY(scale.x, 0);
    if (dragDirection === "y") return new XY(0, scale.y);
    return scale;
  }, [dragScale, dragDirection]);
  const normalDragThreshold = useMemo(
    () => (dragThreshold != null ? new XY(dragThreshold) : null),
    [dragThreshold]
  );

  useVirtualCursorDrag({
    ref: elRef,
    onMove: useCallback(
      (box: Box) => {
        if (elRef.current == null) return;
        let value = vRef.current.prev;
        vRef.current.dragging = true;
        const { x, y } = normalDragThreshold ?? new XY(new Box(elRef.current).dims);
        if (box.width > x && box.width < box.height) {
          const offset = box.signedWidth < 0 ? x : -x;
          value += (box.signedWidth + offset) * normalDragScale.x;
        }
        if (box.height > y && box.height > box.width) {
          const offset = box.signedHeight < 0 ? y : -y;
          value += (box.signedHeight + offset) * normalDragScale.y;
        }
        vRef.current.curr = value;
        onChange(value);
      },
      [onChange, normalDragScale, normalDragThreshold]
    ),
    onEnd: useCallback(() => {
      vRef.current.prev = vRef.current.curr;
      vRef.current.dragging = false;
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

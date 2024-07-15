// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/DragButton.css";

import { Icon } from "@synnaxlabs/media";
import { box,type direction, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Cursor } from "@/cursor";
import { useVirtualCursorDragWebKit } from "@/hooks/useCursorDrag/useVirtualCursorDragWebKit";
import { type Control } from "@/input/types";

export interface DragButtonExtraProps {
  direction?: direction.Crude;
  dragDirection?: direction.Crude;
  dragScale?: xy.Crude | number;
  dragThreshold?: xy.Crude | number;
  resetValue?: number;
}

export interface DragButtonProps
  extends Omit<
      Button.IconProps,
      | "direction"
      | "onChange"
      | "onDragStart"
      | "children"
      | "value"
      | "onDragEnd"
      | "onBlur"
    >,
    Control<number>,
    DragButtonExtraProps {
  onDragEnd?: (value: number) => void;
  onBlur?: () => void;
}

const calculateValue = (
  value: number,
  b: box.Box,
  normalDragScale: xy.XY,
  normalDragThreshold: xy.XY | null,
  elBox: box.Box,
): number => {
  const { x, y } = normalDragThreshold ?? xy.construct(box.dims(elBox));
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
  return value;
};

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
  onDragEnd,
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

  useVirtualCursorDragWebKit({
    ref: elRef,
    onMove: useCallback(
      (b: box.Box) => {
        if (elRef.current == null) return;
        let value = vRef.current.prev;
        vRef.current.dragging = true;
        value = calculateValue(
          value,
          b,
          normalDragScale,
          normalDragThreshold,
          box.construct(elRef.current),
        );
        vRef.current.curr = value;
        onChange(value);
      },
      [onChange, normalDragScale, normalDragThreshold],
    ),
    onEnd: useCallback(
      (b: box.Box, _: unknown) => {
        if (elRef.current == null) return;
        let value = vRef.current.prev;
        value = calculateValue(
          value,
          b,
          normalDragScale,
          normalDragThreshold,
          box.construct(elRef.current),
        );

        vRef.current.prev = value;
        vRef.current.dragging = false;
        Cursor.clearGlobalStyle();
        onDragEnd?.(value);
        props.onBlur?.();
      },
      [props.onBlur, onDragEnd, normalDragScale, normalDragThreshold],
    ),
  });

  const handleDoubleClick = useCallback(() => {
    onChange(resetValue ?? vRef.current.prev);
  }, [onChange, resetValue]);

  return (
    <Button.Icon
      tabIndex={-1}
      ref={elRef}
      variant="outlined"
      className={CSS(CSS.BE("input", "drag-btn"), CSS.dir(direction), className)}
      onDoubleClick={handleDoubleClick}
      onClick={(e) => e.preventDefault()}
      {...props}
    >
      <Icon.Drag />
    </Button.Icon>
  );
};

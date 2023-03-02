// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useMemo, useRef } from "react";

import clsx from "clsx";
import { GrDrag } from "react-icons/gr";

import { InputControl } from "./types";

import { Button, ButtonIconProps } from "@/core/Button";
import { directionCls } from "@/css";
import { Box, Direction, toXY, XY } from "@/spatial";
import { useVirtualCursorDrag } from "@/spatial/useCursorDrag";

export interface InputDragButtonExtensionProps {
  direction?: Direction;
  dragDirection?: Direction;
  dragScale?: XY | number;
  dragThreshold?: XY | number;
}

export interface InputDragButtonProps
  extends Omit<ButtonIconProps, "direction" | "onChange" | "onDragStart" | "children">,
    InputControl<number>,
    InputDragButtonExtensionProps {}

export const InputDragButton = ({
  direction,
  className,
  dragScale = { x: 10, y: 1 },
  dragThreshold = 12,
  dragDirection,
  onChange,
  value,
  ...props
}: InputDragButtonProps): JSX.Element => {
  const vRef = useRef({
    dragging: false,
    curr: value,
    prev: value,
  });
  const elRef = useRef<HTMLButtonElement>(null);

  if (!vRef.current.dragging) vRef.current.prev = value;

  const normalDragScale = useMemo(() => {
    const scale = toXY(dragScale);
    if (dragDirection === "x") scale.y = 0;
    else if (dragDirection === "y") scale.x = 0;
    return scale;
  }, [dragScale, dragDirection]);
  const normalDragThreshold = useMemo(() => toXY(dragThreshold ?? 0), [dragThreshold]);

  useVirtualCursorDrag({
    ref: elRef,
    onMove: useCallback(
      (box: Box) => {
        let value = vRef.current.prev;
        vRef.current.dragging = true;
        const { x, y } = normalDragThreshold;
        if (box.width > x) value += (box.signedWidth - x) * normalDragScale.x;
        if (box.height > y) value += (box.signedHeight - y) * normalDragScale.y;
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

  return (
    <Button.Icon
      ref={elRef}
      variant="outlined"
      className={clsx(
        "pluto-input-number__drag-button",
        direction != null && directionCls(direction),
        className
      )}
      onDoubleClick={() => onChange(vRef.current.prev)}
      {...props}
    >
      <GrDrag />
    </Button.Icon>
  );
};

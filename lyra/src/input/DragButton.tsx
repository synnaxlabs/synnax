// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/DragButton.css";

import { box, type direction, xy } from "@synnaxlabs/x";
import { type ReactElement, useRef } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Cursor } from "@/cursor";
import { Icon } from "@/icon";
import { type Control } from "@/input/types";
import { preventDefault } from "@/util";

/** Drag behavior an input passes down to its {@link DragButton}. */
export interface DragButtonExtraProps {
  direction?: direction.Crude;
  /** Restricts scrubbing to one axis. Both axes are live when unset. */
  dragDirection?: direction.Crude;
  /** Value change per pixel dragged, per axis. */
  dragScale?: xy.Crude | number;
  /** Pixels the pointer must travel before scrubbing starts. */
  dragThreshold?: xy.Crude | number;
  /** Value a double click restores. */
  resetValue?: number;
}

export interface DragButtonProps
  extends
    Omit<
      Button.ButtonProps,
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

/**
 * A handle that scrubs a number as the pointer drags across it, horizontally by the x
 * scale and vertically by the y. A double click restores `resetValue`.
 */
export const DragButton = ({
  direction,
  className,
  dragScale = { x: 10, y: 1 },
  dragThreshold = 15,
  dragDirection,
  onChange,
  value,
  resetValue,
  onDragEnd,
  disabled,
  ...rest
}: DragButtonProps): ReactElement => {
  // The value a drag scrubs from, held until the drag ends.
  const startRef = useRef<number | null>(null);
  const elRef = useRef<HTMLButtonElement>(null);

  const scale = xy.construct(dragScale);
  let normalDragScale = scale;
  if (dragDirection === "x") normalDragScale = xy.construct(scale.x, 0);
  else if (dragDirection === "y") normalDragScale = xy.construct(0, scale.y);
  const normalDragThreshold =
    dragThreshold != null ? xy.construct(dragThreshold) : null;

  const scrub = (b: box.Box, el: HTMLElement): number => {
    if (startRef.current == null) startRef.current = value;
    return calculateValue(
      startRef.current,
      b,
      normalDragScale,
      normalDragThreshold,
      box.construct(el),
    );
  };

  Cursor.useVirtualDrag({
    ref: elRef,
    onMove: (b: box.Box) => {
      if (elRef.current == null || disabled) return;
      onChange(scrub(b, elRef.current));
    },
    onEnd: (b: box.Box) => {
      if (elRef.current == null || disabled) return;
      const next = scrub(b, elRef.current);
      startRef.current = null;
      Cursor.clearGlobalStyle();
      onDragEnd?.(next);
      rest.onBlur?.();
    },
  });

  const handleDoubleClick = () => onChange(resetValue ?? value);

  return (
    <Button.Button
      ref={elRef}
      variant="outlined"
      className={CSS.cls(
        CSS.BE("input", "drag-btn"),
        direction != null && CSS.BEM("input", "drag-btn", "direction", direction),
        className,
      )}
      tabIndex={-1}
      onDoubleClick={handleDoubleClick}
      onClick={preventDefault}
      textColor={9}
      disabled={disabled}
      {...rest}
    >
      <Icon.Drag />
    </Button.Button>
  );
};

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatch.css";

import { useCallback } from "react";

import { Button } from "@/button";
import { color } from "@/color/core";
import { type Color, type Crude } from "@/color/core/color";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { Theming } from "@/theming";

const HAUL_TYPE = "color";

export interface BaseSwatchProps
  extends Omit<Button.ButtonProps, "onChange" | "value" | "size"> {
  value: Crude;
  onChange?: (c: Color) => void;
  size?: Button.ButtonProps["size"] | "tiny";
}

export const BaseSwatch = ({
  value,
  onChange,
  className,
  size = "medium",
  draggable = true,
  style,
  ...rest
}: BaseSwatchProps) => {
  const background = Theming.use().colors.gray.l0;
  const clr = new color.Color(value);
  const dragging = Haul.useDraggingState();
  const canDrop: Haul.CanDrop = useCallback(
    ({ items }) => {
      const [k] = Haul.filterByType(HAUL_TYPE, items);
      return k != null && k.key !== clr.hex;
    },
    [clr.hex],
  );
  const handleDrop: Haul.OnDrop = useCallback(
    ({ items }) => {
      const [k] = Haul.filterByType(HAUL_TYPE, items);
      if (k != null) onChange?.(new color.Color(k.key as string));
      return items;
    },
    [onChange],
  );
  const { startDrag, ...haulProps } = Haul.useDragAndDrop({
    type: "Color.Swatch",
    onDrop: handleDrop,
    canDrop,
  });
  const handleDragStart = useCallback(() => {
    startDrag([{ type: HAUL_TYPE, key: clr.hex }]);
  }, [startDrag, clr.hex]);
  return (
    <Button.Button
      className={CSS(
        CSS.B("color-swatch"),
        CSS.M(size),
        clr.contrast(background) > 1.5 && clr.a > 0.5 && CSS.M("no-border"),
        CSS.dropRegion(canDrop(dragging)),
        className,
      )}
      size={size as Button.ButtonProps["size"]}
      draggable={draggable}
      onDragStart={handleDragStart}
      style={{ ...style, backgroundColor: color.cssString(value) }}
      variant="text"
      {...haulProps}
      {...rest}
    />
  );
};

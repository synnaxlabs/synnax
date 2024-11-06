// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatch/Swatch.css";

import { type ReactElement, useCallback } from "react";

import { Button } from "@/button";
import { color } from "@/color/core";
import { type Color, type Crude } from "@/color/core/color";
import { Picker } from "@/color/Picker";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { type UseProps } from "@/dropdown/Dropdown";
import { Haul } from "@/haul";
import { type Input } from "@/input";
import { Text } from "@/text";
import { Theming } from "@/theming";

export interface SwatchProps
  extends Input.Control<Crude, Color>,
    Omit<Button.ButtonProps, "onChange" | "value">,
    UseProps {
  allowChange?: boolean;
}

const HAUL_TYPE = "color";

export const Swatch = ({
  value,
  onChange,
  className,
  size = "medium",
  onVisibleChange,
  initialVisible,
  allowChange = true,
  draggable = true,
  ...props
}: SwatchProps): ReactElement => {
  const { visible, open, close } = Dropdown.use({ onVisibleChange, initialVisible });

  const bg = Theming.use().colors.gray.l0;

  const d = new color.Color(value);

  const dragging = Haul.useDraggingState();

  const canDrop: Haul.CanDrop = useCallback(
    ({ items }) => {
      const [k] = Haul.filterByType(HAUL_TYPE, items);
      return k != null && k.key !== d.hex;
    },
    [d.hex],
  );

  const { startDrag, ...haulProps } = Haul.useDragAndDrop({
    type: "Color.Swatch",
    onDrop: ({ items }) => {
      const dropped = Haul.filterByType(HAUL_TYPE, items);
      if (items.length > 0) onChange?.(new color.Color(dropped[0].key as string));
      return dropped;
    },
    canDrop,
  });

  const canChange = onChange != null && allowChange;

  const swatch = (
    <Button.Button
      className={CSS(
        CSS.B("color-swatch"),
        CSS.size(size),
        d.contrast(bg) > 1.5 && d.a > 0.5 && CSS.M("no-border"),
        CSS.dropRegion(canDrop(dragging)),
        className,
      )}
      disabled={!canChange}
      draggable={draggable}
      onDragStart={() => startDrag([{ type: HAUL_TYPE, key: d.hex }])}
      style={{ backgroundColor: color.cssString(value) }}
      variant="text"
      onClick={open}
      size={size}
      tooltip={
        canChange ? (
          <Text.Text level="small">Click to change color</Text.Text>
        ) : undefined
      }
      {...haulProps}
      {...props}
    />
  );

  if (!canChange) return swatch;

  return (
    <Dropdown.Dialog
      close={close}
      visible={visible}
      className={CSS.BE("color-swatch", "dropdown")}
      keepMounted={false}
      variant="floating"
      zIndex={100}
    >
      {swatch}
      <Picker value={value} onChange={onChange} />
    </Dropdown.Dialog>
  );
};

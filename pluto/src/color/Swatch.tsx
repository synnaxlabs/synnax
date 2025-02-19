// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatch.css";

import { type ReactElement, useCallback } from "react";

import { Button } from "@/button";
import { color } from "@/color/core";
import { type Color, type Crude } from "@/color/core/color";
import { Picker, type PickerProps } from "@/color/Picker";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { type UseProps } from "@/dropdown/Dropdown";
import { Haul } from "@/haul";
import { Text } from "@/text";
import { Theming } from "@/theming";

export interface SwatchProps
  extends Omit<Button.ButtonProps, "onChange" | "value" | "size">,
    UseProps,
    Pick<PickerProps, "onDelete" | "position"> {
  allowChange?: boolean;
  value: Crude;
  onChange?: (c: Color) => void;
  size?: Button.ButtonProps["size"] | "tiny";
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
  style,
  onClick,
  ...rest
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

  const canPick = onChange != null && allowChange;

  const swatch = (
    <Button.Button
      className={CSS(
        CSS.B("color-swatch"),
        CSS.M(size),
        d.contrast(bg) > 1.5 && d.a > 0.5 && CSS.M("no-border"),
        CSS.dropRegion(canDrop(dragging)),
        className,
      )}
      disabled={!canPick && onClick == null}
      size={size as Button.ButtonProps["size"]}
      draggable={draggable}
      onDragStart={() => startDrag([{ type: HAUL_TYPE, key: d.hex }])}
      style={{ backgroundColor: color.cssString(value) }}
      variant="text"
      onClick={canPick ? open : onClick}
      tooltip={
        canPick ? <Text.Text level="small">Click to change color</Text.Text> : undefined
      }
      {...haulProps}
      {...rest}
    />
  );

  if (!canPick) return swatch;

  return (
    <Dropdown.Dialog
      close={close}
      visible={visible}
      className={CSS.BE("color-swatch", "dropdown")}
      keepMounted={false}
      variant="floating"
      zIndex={100}
      style={style}
    >
      {swatch}
      <Picker value={value} onChange={onChange} />
    </Dropdown.Dialog>
  );
};

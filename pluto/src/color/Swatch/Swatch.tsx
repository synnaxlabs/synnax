// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback } from "react";

import { color } from "@/color/core";
import { Picker } from "@/color/Picker";
import { Button, ButtonProps, Dropdown, InputControl, Text } from "@/core/std";
import { UseProps } from "@/dropdown/Dropdown";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { Theming } from "@/theming/main";

import "@/color/Swatch/Swatch.css";

export interface SwatchProps
  extends InputControl<color.Crude, color.Color>,
    Omit<ButtonProps, "onChange" | "value">,
    UseProps {}

const HAUL_TYPE = "color";

export const ColorSwatch = ({
  value,
  onChange,
  className,
  size = "medium",
  onVisibleChange,
  initialVisible,
  ...props
}: SwatchProps): ReactElement => {
  const { visible, open, ref } = Dropdown.use({ onVisibleChange, initialVisible });

  const bg = Theming.use().colors.background;

  const d = new color.Color(value);

  const { startDrag, endDrag } = Haul.useDrag();

  const dragging = Haul.useDraggingState();

  const canDrop = useCallback(
    (dragging: Haul.Item[]) => {
      const k = dragging.find((i) => i.type === HAUL_TYPE)?.key;
      if (k == null) return false;
      return k !== d.hex;
    },
    [d.hex]
  );

  const dropProps = Haul.useDrop({
    onDrop: (item) => {
      const k = item.find((i) => i.type === HAUL_TYPE)?.key;
      if (k == null) return;
      onChange?.(new color.Color(k as string));
    },
    canDrop,
  });

  const swatch = (
    <Button
      className={CSS(
        CSS.B("color-swatch"),
        CSS.size(size),
        d.contrast(bg) > 1.5 && d.a > 0.5 && CSS.M("no-border"),
        CSS.dropRegion(canDrop(dragging)),
        className
      )}
      draggable
      onDragStart={() =>
        startDrag([
          {
            type: "color",
            key: d.hex,
          },
        ])
      }
      {...dropProps}
      onDragEnd={endDrag}
      style={{ backgroundColor: color.cssString(value) }}
      variant="text"
      onClick={open}
      size={size}
      tooltip={
        onChange != null ? <Text level="small">Click to change color</Text> : undefined
      }
      {...props}
    />
  );

  if (onChange == null) return swatch;

  return (
    <Dropdown
      visible={visible}
      ref={ref}
      className={CSS.BE("color-swatch", "dropdown")}
      keepMounted={false}
    >
      {swatch}
      <Picker value={value} onChange={onChange} />
    </Dropdown>
  );
};

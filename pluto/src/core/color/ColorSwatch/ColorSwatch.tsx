// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Color, ColorT } from "@/core/color/color";
import { ColorPicker } from "@/core/color/ColorPicker";
import { CSS } from "@/core/css";
import { Button, ButtonProps, Dropdown, InputControl } from "@/core/std";
import { UseDropdownProps } from "@/core/std/Dropdown/Dropdown";

import "@/core/color/ColorSwatch/ColorSwatch.css";

export interface ColorSwatchProps
  extends InputControl<ColorT, Color>,
    Omit<ButtonProps, "onChange" | "value">,
    UseDropdownProps {}

export const ColorSwatch = ({
  value,
  onChange,
  className,
  size = "medium",
  onVisibleChange,
  initialVisible,
  ...props
}: ColorSwatchProps): ReactElement => {
  const { visible, open, ref } = Dropdown.use({ onVisibleChange, initialVisible });
  const color = new Color(value);

  const swatch = (
    <Button
      className={CSS(CSS.B("color-swatch"), CSS.size(size), className)}
      style={{ backgroundColor: color.hex }}
      variant="text"
      onClick={open}
      size={size}
      {...props}
    />
  );

  if (onChange == null) return swatch;

  return (
    <Dropdown
      visible={visible}
      ref={ref}
      className={CSS.BE("color-swatch", "dropdown")}
    >
      {swatch}
      <ColorPicker value={value} onChange={onChange} />
    </Dropdown>
  );
};

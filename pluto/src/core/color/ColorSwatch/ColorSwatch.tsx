// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Color } from "@/core/color/color";
import { ColorPicker } from "@/core/color/ColorPicker";
import { CSS } from "@/core/css";
import { Button, ButtonProps, Dropdown, InputControl } from "@/core/std";

import "@/core/color/ColorSwatch/ColorSwatch.css";

export interface ColorSwatchProps
  extends InputControl<Color>,
    Omit<ButtonProps, "onChange" | "value"> {}

export const ColorSwatch = ({
  value,
  onChange,
  className,
  size = "medium",
  ...props
}: ColorSwatchProps): ReactElement => {
  const { visible, open, ref } = Dropdown.use();

  const swatch = (
    <Button
      className={CSS(CSS.B("color-swatch"), CSS.size(size), className)}
      style={{ backgroundColor: value.hex }}
      variant="text"
      onClick={open}
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

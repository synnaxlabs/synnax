// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Picker/Picker.css";

import { type ComponentPropsWithoutRef, type ReactElement } from "react";
import { type ColorResult,SketchPicker } from "react-color";

import { color } from "@/color/core";
import { CSS } from "@/css";
import { type Input } from "@/input";

export interface PickerProps
  extends Input.Control<color.Crude, color.Color>,
    Omit<ComponentPropsWithoutRef<"div">, "onChange"> {}

export const Picker = ({ value, onChange, ...props }: PickerProps): ReactElement => {
  const handleChange = (res: ColorResult): void => {
    if (res.hex === "transparent") onChange(color.ZERO);
    onChange(new color.Color(res.hex, res.rgb.a));
  };

  return (
    <SketchPicker
      className={CSS.B("color-picker")}
      color={new color.Color(value).hex}
      onChange={handleChange}
      presetColors={[]}
      {...props}
    />
  );
};

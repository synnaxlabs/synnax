// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Picker/Picker.css";

import { Icon } from "@synnaxlabs/media";
import { type ComponentPropsWithoutRef, type ReactElement } from "react";
import { type ColorResult, SketchPicker } from "react-color";

import { Align } from "@/align";
import { Button } from "@/button";
import { color } from "@/color/core";
import { CSS } from "@/css";
import { type Input } from "@/input";
import { Text } from "@/text";

export interface PickerProps
  extends Input.Control<color.Crude, color.Color>,
    Omit<ComponentPropsWithoutRef<"div">, "onChange"> {
  onDelete?: () => void;
  position?: number;
}

export const Picker = ({
  value,
  onChange,
  position,
  onDelete,
  ...props
}: PickerProps): ReactElement => {
  const handleChange = (res: ColorResult): void => {
    if (res.hex === "transparent") onChange(color.ZERO);
    onChange(new color.Color(res.hex, res.rgb.a));
  };

  return (
    <Align.Space direction="y">
      {position != null ||
        (onDelete != null && (
          <Align.Space direction="x" justify="spaceBetween">
            {position != null && (
              <Text.Text level="small" shade={7}>
                {position} %
              </Text.Text>
            )}
            {onDelete != null && (
              <Button.Icon name="close" onClick={onDelete} size="small">
                <Icon.Delete />
              </Button.Icon>
            )}
          </Align.Space>
        ))}

      <SketchPicker
        className={CSS.B("color-picker")}
        color={new color.Color(value).hex}
        onChange={handleChange}
        presetColors={[]}
        {...props}
      />
    </Align.Space>
  );
};

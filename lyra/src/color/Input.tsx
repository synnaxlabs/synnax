// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Input.css";

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { HexText, type HexTextProps } from "@/color/HexText";
import { Swatch, type SwatchProps } from "@/color/Swatch";
import { CSS } from "@/css";
import { Flex } from "@/flex";

export interface InputProps
  extends
    Omit<HexTextProps, "text" | "alpha" | "onChange" | "onClear" | "startContent">,
    Pick<SwatchProps, "mixed"> {
  /** The picked color. Absent means the theme picks. */
  value?: color.Crude;
  /** Called with the new color, or with undefined when the user picks Auto. */
  onChange: (value?: color.Color) => void;
  /** The color the theme paints while the value is absent. */
  fallback: color.Crude;
}

/**
 * An optional color as a swatch joined to a hex text box. The user types or pastes a
 * color into the box, or clicks the swatch to open the picker. An empty box means
 * Auto: the swatch shows the fallback, and the box shows "Auto". While mixed, the box
 * shows "Mixed" until the user sets a color.
 */
export const Input = ({
  value,
  onChange,
  fallback,
  onlyChangeOnBlur,
  mixed = false,
  size = "medium",
  className,
  ...rest
}: InputProps): ReactElement => (
  <Flex.Box pack className={CSS.cls(CSS.B("color-input"), className)}>
    <Swatch
      value={value}
      fallback={fallback}
      onChange={onChange}
      onlyChangeOnBlur={onlyChangeOnBlur}
      mixed={mixed}
      size={size}
    />
    <HexText
      size={size}
      placeholder={mixed ? "Mixed" : "Auto"}
      startContent={mixed || value == null ? undefined : "#"}
      text={mixed || value == null ? "" : color.hex(value).slice(1)}
      alpha={color.aValue(value ?? fallback)}
      onChange={onChange}
      onClear={() => onChange(undefined)}
      onlyChangeOnBlur={onlyChangeOnBlur}
      {...rest}
    />
  </Flex.Box>
);

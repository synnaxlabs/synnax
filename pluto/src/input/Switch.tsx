// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Switch.css";

import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { type InputProps } from "@/input/types";

export interface SwitchProps
  extends InputProps<boolean>,
    Omit<Button.ExtensionProps, "variant"> {}

/**
 * A controlled boolean Switch input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @default "medium"
 */
export const Switch = ({
  ref,
  className,
  value,
  disabled,
  onChange,
  size,
  variant,
  style,
  color,
  borderColor,
  borderWidth,
  bordered,
  rounded,
  background,
  ...rest
}: SwitchProps): ReactElement => (
  <Button.Button
    el="label"
    variant="text"
    className={CSS(CSS.BE("input", "switch"), className)}
    disabled={disabled}
    size={size}
    preventClick
    style={style}
    color={color}
    borderColor={borderColor}
    borderWidth={borderWidth}
    bordered={bordered}
    rounded={rounded}
    background={background}
  >
    <input
      type="checkbox"
      ref={ref}
      checked={value}
      onChange={(e) => onChange(e.target.checked)}
      value=""
      disabled={disabled}
      {...rest}
    />
    <span className={CSS(CSS.BE("input", "switch", "track"))} />
  </Button.Button>
);

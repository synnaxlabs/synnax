// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { type InputProps } from "@/input/types";

export interface BooleanProps
  extends Omit<InputProps<boolean>, "onClick">, Omit<Button.ExtensionProps, "variant"> {
  inputType: "switch" | "checkbox";
  onClick?: (e: React.MouseEvent<HTMLElement>) => void;
  /** When set, replaces the box indicator with this glyph in the checked state. */
  checkedIcon?: ReactNode;
  /** Glyph shown in the unchecked state when {@link checkedIcon} is set. */
  uncheckedIcon?: ReactNode;
}

const parseTextColor = (
  preview: boolean | undefined,
  textColor: Button.ButtonProps["color"],
  value: boolean,
): Button.ButtonProps["color"] => {
  if (preview === true && value === true) return "var(--pluto-primary-z)";
  return textColor;
};

/**
 * Base Boolean input component for switch and checkbox variants.
 */
export const Boolean = ({
  ref,
  className,
  value,
  reveal,
  disabled,
  onChange,
  inputType,
  size,
  variant = "text",
  preview,
  style,
  color,
  borderColor,
  borderWidth,
  bordered,
  rounded,
  background,
  textColor,
  onClick,
  checkedIcon,
  uncheckedIcon,
  tooltip,
  tooltipLocation,
  hideTooltip,
  ...rest
}: BooleanProps): ReactElement => (
  <Button.Button
    el="label"
    variant={variant === "shadow" ? "outlined" : variant}
    className={CSS(
      CSS.BE("input", inputType),
      checkedIcon != null && CSS.M("icon"),
      className,
    )}
    reveal={reveal}
    disabled={disabled}
    size={size}
    preview={preview}
    preventClick
    style={style}
    color={color}
    borderColor={borderColor}
    borderWidth={borderWidth}
    bordered={bordered}
    rounded={rounded}
    background={background}
    textColor={parseTextColor(preview, textColor, value)}
    onClick={onClick}
    tooltip={tooltip}
    tooltipLocation={tooltipLocation}
    hideTooltip={hideTooltip}
  >
    {preview !== true ? (
      <>
        <input
          className={CSS.BE("input", inputType, "input")}
          type="checkbox"
          ref={ref}
          checked={value}
          onChange={(e) => {
            e.stopPropagation();
            onChange?.(e.target.checked);
          }}
          value=""
          disabled={disabled}
          onClick={onClick}
          {...rest}
        />
        <span className={CSS.BE("input", inputType, "indicator")} onClick={onClick}>
          {value ? checkedIcon : uncheckedIcon}
        </span>
      </>
    ) : value ? (
      "True"
    ) : (
      "False"
    )}
  </Button.Button>
);

// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Input.css";

import { forwardRef, useRef, useState } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useCombinedRefs } from "@/hooks";
import { type BaseProps } from "@/input/types";
import { Status } from "@/status";
import { Text as BaseText, Text as CoreText } from "@/text";

export interface TextExtraProps {
  selectOnFocus?: boolean;
  centerPlaceholder?: boolean;
  resetOnBlurIfEmpty?: boolean;
  status?: Status.Variant;
  onlyChangeOnBlur?: boolean;
  shade?: BaseText.Shade;
  weight?: BaseText.Weight;
}

export interface TextProps extends BaseProps<string>, TextExtraProps {}

/**
 * A controlled string input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @param props.selectOnFocus - Whether the input should select its contents when focused.
 * @param props.centerPlaceholder - Whether the placeholder should be centered.
 * @param props.resetOnBlurIfEmpty - Whether the input should reset to its previous value if
 * blurred while empty.
 */
export const Text = forwardRef<HTMLInputElement, TextProps>(
  (
    {
      size = "medium",
      value,
      style,
      onChange,
      className,
      onFocus,
      selectOnFocus = false,
      centerPlaceholder = false,
      placeholder,
      variant = "outlined",
      sharp = false,
      children,
      level,
      onBlur,
      disabled,
      resetOnBlurIfEmpty = false,
      status,
      shade,
      weight,
      onlyChangeOnBlur = false,
      ...props
    },
    ref,
  ) => {
    const cachedFocusRef = useRef("");
    const [tempValue, setTempValue] = useState<string | null>(null);

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>): void => {
      if (resetOnBlurIfEmpty && e.target.value === "")
        onChange?.(cachedFocusRef.current);
      else if (onlyChangeOnBlur) {
        console.log(tempValue);
        if (tempValue != null) onChange?.(tempValue);
      }
      setTempValue(null);
      onBlur?.(e);
    };

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
      if (!onlyChangeOnBlur) onChange?.(e.target.value);
      else setTempValue(e.target.value);
    };

    const handleFocus = (e: React.FocusEvent<HTMLInputElement>): void => {
      if (onlyChangeOnBlur) setTempValue(value);
      onFocus?.(e);
      cachedFocusRef.current = e.target.value;
      // This looks hacky, but it's the only way to consistently select the text
      // after the focus event.
      if (!selectOnFocus) return;
      const interval = setInterval(() => e.target.select(), 2);
      setTimeout(() => clearInterval(interval), 50);
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>): void => {
      if (!onlyChangeOnBlur) return;
      if (e.key === "Enter") e.currentTarget.blur();
    };

    const internalRef = useRef<HTMLInputElement>(null);
    const combinedRef = useCombinedRefs(ref, internalRef);

    return (
      <Align.Pack
        style={style}
        className={CSS(
          CSS.B("input"),
          disabled && CSS.BM("input", "disabled"),
          level == null && CSS.size(size),
          shade != null && CSS.shade(shade),
          CSS.BM("input", variant),
          CSS.sharp(sharp),
          status != null && CSS.M(status),
          className,
        )}
        align="center"
        size={size}
      >
        <div className={CSS.BE("input", "internal")}>
          {(value == null || value.length === 0) && tempValue == null && (
            <div
              className={CSS(
                CSS.BE("input", "placeholder"),
                centerPlaceholder && CSS.M("centered"),
              )}
            >
              {CoreText.formatChildren(
                level ?? CoreText.ComponentSizeLevels[size],
                placeholder,
              )}
            </div>
          )}
          <input
            ref={combinedRef}
            value={tempValue ?? value}
            onChange={handleChange}
            role="textbox"
            autoCapitalize="off"
            autoComplete="off"
            autoCorrect="off"
            onFocus={handleFocus}
            onKeyDown={handleKeyDown}
            onBlur={handleBlur}
            className={CSS(CSS.visible(false), level != null && CSS.BM("text", level))}
            disabled={disabled}
            placeholder={typeof placeholder === "string" ? placeholder : undefined}
            style={{ fontWeight: weight }}
            {...props}
          />
        </div>
        {children}
      </Align.Pack>
    );
  },
);
Text.displayName = "Input";

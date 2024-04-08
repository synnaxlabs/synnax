// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, useRef } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type BaseProps } from "@/input/types";
import { Text as CoreText } from "@/text";

import "@/input/Input.css";

export interface TextExtraProps {
  selectOnFocus?: boolean;
  centerPlaceholder?: boolean;
  resetOnBlurIfEmpty?: boolean;
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
      resetOnBlurIfEmpty = false,
      ...props
    },
    ref,
  ) => {
    const cachedFocusRef = useRef("");

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>): void => {
      if (resetOnBlurIfEmpty && e.target.value === "")
        onChange?.(cachedFocusRef.current);
      onBlur?.(e);
    };

    return (
      <Align.Pack
        style={style}
        className={CSS(
          CSS.B("input"),
          level == null && CSS.size(size),
          CSS.BM("input", variant),
          CSS.sharp(sharp),
          className,
        )}
        align="center"
        size={size}
      >
        <div className={CSS.BE("input", "internal")}>
          {(value == null || value.length === 0) && (
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
            ref={ref}
            value={value}
            onChange={(e) => {
              onChange?.(e.target.value);
            }}
            role="textbox"
            autoCapitalize="off"
            autoComplete="off"
            autoCorrect="off"
            onFocus={(e) => {
              onFocus?.(e);
              cachedFocusRef.current = e.target.value;
              if (selectOnFocus) setTimeout(() => e.target.select(), 2);
            }}
            onBlur={handleBlur}
            className={CSS(CSS.visible(false), level != null && CSS.BM("text", level))}
            {...props}
          />
        </div>
        {children}
      </Align.Pack>
    );
  },
);
Text.displayName = "Input";

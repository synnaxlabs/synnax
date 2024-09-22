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
import { Text as CoreText } from "@/text";

export interface TextExtraProps {
  selectOnFocus?: boolean;
  centerPlaceholder?: boolean;
  resetOnBlurIfEmpty?: boolean;
  status?: Status.Variant;
  onlyChangeOnBlur?: boolean;
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
      color,
      onlyChangeOnBlur = false,
      endContent,
      ...props
    },
    ref,
  ) => {
    const cachedFocusRef = useRef("");
    const [tempValue, setTempValue] = useState<string | null>(null);
    const internalRef = useRef<HTMLInputElement>(null);
    const focusedRef = useRef(false);

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>): void => {
      focusedRef.current = false;
      if (resetOnBlurIfEmpty && e.target.value === "")
        onChange?.(cachedFocusRef.current);
      else if (onlyChangeOnBlur) {
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
    };

    const handleMouseUp = (): void => {
      // This looks hacky, but it's the only way to consistently select the text
      // after the focus event.
      if (!selectOnFocus || focusedRef.current) return;
      focusedRef.current = true;
      internalRef.current?.select();
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>): void => {
      if (!onlyChangeOnBlur) return;
      if (e.key === "Enter") e.currentTarget.blur();
    };

    const combinedRef = useCombinedRefs(ref, internalRef);

    const disabledCSS = disabled && CSS.BM("input", "disabled");
    if (variant === "preview") disabled = true;

    const showPlaceholder = (value == null || value.length === 0) && tempValue == null;

    return (
      <Align.Pack
        style={style}
        className={CSS(
          CSS.B("input"),
          disabledCSS,
          level == null && CSS.size(size),
          shade != null && CSS.shade(shade),
          CSS.BM("input", variant),
          CSS.sharp(sharp),
          status != null && CSS.M(status),
          className,
        )}
        borderShade={4}
        align="center"
        size={size}
      >
        <div className={CSS.BE("input", "internal")}>
          {showPlaceholder && (
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
          {endContent != null && (
            <div className={CSS.BE("input", "end-content")}>
              {CoreText.formatChildren(
                level ?? CoreText.ComponentSizeLevels[size],
                endContent,
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
            onMouseUp={handleMouseUp}
            onBlur={handleBlur}
            className={CSS(CSS.visible(false), level != null && CSS.BM("text", level))}
            disabled={disabled}
            placeholder={typeof placeholder === "string" ? placeholder : undefined}
            style={{ fontWeight: weight, color }}
            {...props}
          />
        </div>
        {children}
      </Align.Pack>
    );
  },
);
Text.displayName = "Input";

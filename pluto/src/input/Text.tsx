// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Input.css";

import { type status } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useRef, useState } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Generic } from "@/generic";
import { useCombinedRefs } from "@/hooks";
import { type InputProps, type Variant } from "@/input/types";
import { Text as BaseText } from "@/text";
import { type Tooltip } from "@/tooltip";

export interface TextProps
  extends
    InputProps<string>,
    Omit<Button.ExtensionProps, "variant">,
    Tooltip.WrapProps {
  selectOnFocus?: boolean;
  centerPlaceholder?: boolean;
  resetOnBlurIfEmpty?: boolean;
  status?: status.Variant;
  variant?: Variant;
  placeholder?: ReactNode;
  children?: ReactNode;
  endContent?: ReactNode;
  startContent?: ReactNode;
  onlyChangeOnBlur?: boolean;
  area?: boolean;
}

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
 * @param props.onlyChangeOnBlur - If true, the input will only call `onChange` when the
 * user blurs the input or the user presses 'Enter'.
 */
export const Text = ({
  size = "medium",
  ref,
  value,
  onChange,
  className,
  onFocus,
  onKeyDown,
  selectOnFocus = false,
  centerPlaceholder = false,
  placeholder,
  variant = "outlined",
  level,
  onBlur,
  disabled,
  resetOnBlurIfEmpty = false,
  status,
  weight,
  style,
  contrast,
  color: pColor,
  sharp,
  onlyChangeOnBlur = false,
  endContent,
  full,
  children,
  grow,
  shrink,
  borderColor,
  borderWidth,
  bordered,
  rounded,
  tabIndex,
  trigger,
  triggerIndicator,
  textColor,
  textVariant,
  preventClick,
  onClickDelay,
  startContent,
  tooltip,
  tooltipDelay,
  tooltipLocation,
  hideTooltip,
  ghost,
  area,
  propagateClick,
  ...rest
}: TextProps): ReactElement => {
  const cachedFocusRef = useRef(value);
  const [tempValue, setTempValue] = useState<string | null>(null);
  const internalRef = useRef<HTMLInputElement>(null);
  const focusedRef = useRef(false);

  const handleBlur = (e: React.FocusEvent<HTMLInputElement>): void => {
    focusedRef.current = false;
    if (resetOnBlurIfEmpty && e.target.value === "") onChange?.(cachedFocusRef.current);
    else if (onlyChangeOnBlur) if (tempValue != null) onChange?.(tempValue);
    setTempValue(null);
    onBlur?.(e);
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
    if (disabled) return;
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
    if (onlyChangeOnBlur && e.key === "Enter") e.currentTarget.blur();
    else if (e.key === "Escape") {
      e.currentTarget.blur();
      e.stopPropagation();
    }
    onKeyDown?.(e);
  };

  const combinedRef = useCombinedRefs(ref, internalRef);

  const showPlaceholder =
    (value == null || value.length === 0) &&
    tempValue == null &&
    placeholder != null &&
    typeof placeholder !== "string";

  tabIndex ??= variant === "preview" ? -1 : undefined;

  const outerProps: Flex.BoxProps = {
    style,
    full,
    grow,
    shrink,
  };
  const hasChildren = children != null;
  const restButtonProps = hasChildren ? {} : outerProps;

  const baseInput = (
    <Button.Button
      el="div"
      x
      empty
      align="center"
      className={CSS(
        CSS.B("input"),
        CSS.disabled(disabled),
        status != null && CSS.M(status),
        className,
      )}
      size={size}
      level={level}
      color={pColor}
      contrast={contrast}
      sharp={sharp}
      status={status}
      bordered={bordered}
      borderColor={borderColor}
      borderWidth={borderWidth}
      pack
      variant={variant}
      rounded={rounded}
      tabIndex={tabIndex}
      trigger={trigger}
      triggerIndicator={triggerIndicator}
      textColor={textColor}
      textVariant={textVariant}
      preventClick={preventClick}
      onClickDelay={onClickDelay}
      tooltip={tooltip}
      tooltipDelay={tooltipDelay}
      tooltipLocation={tooltipLocation}
      hideTooltip={hideTooltip}
      ghost={ghost}
      propagateClick={propagateClick}
      {...restButtonProps}
    >
      {showPlaceholder && (
        <BaseText.Text
          className={CSS(
            CSS.visible(false),
            CSS.BE("input", "placeholder"),
            centerPlaceholder && CSS.M("centered"),
          )}
          level={level ?? BaseText.COMPONENT_SIZE_LEVELS[size]}
        >
          {placeholder}
        </BaseText.Text>
      )}
      {startContent != null && (
        <BaseText.Text
          className={CSS.BE("input", "start-content")}
          level={level ?? BaseText.COMPONENT_SIZE_LEVELS[size]}
        >
          {startContent}
        </BaseText.Text>
      )}
      <Generic.Element<"textarea" | "input">
        el={area ? "textarea" : "input"}
        ref={combinedRef}
        value={tempValue ?? value}
        role="textbox"
        onChange={handleChange}
        autoCapitalize="off"
        autoComplete="off"
        autoCorrect="off"
        onFocus={handleFocus}
        onKeyDown={handleKeyDown}
        tabIndex={tabIndex}
        onMouseUp={handleMouseUp}
        onBlur={handleBlur}
        disabled={disabled}
        placeholder={typeof placeholder === "string" ? placeholder : undefined}
        style={{ fontWeight: weight }}
        {...rest}
      />
      {endContent != null && (
        <BaseText.Text
          className={CSS.BE("input", "end-content")}
          level={level ?? BaseText.COMPONENT_SIZE_LEVELS[size]}
        >
          {endContent}
        </BaseText.Text>
      )}
    </Button.Button>
  );
  if (children == null) return baseInput;
  return (
    <Flex.Box x pack className={CSS.BE("input", "container")} {...outerProps}>
      {baseInput}
      {children}
    </Flex.Box>
  );
};

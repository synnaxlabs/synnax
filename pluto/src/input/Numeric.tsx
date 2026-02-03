// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x";
import { evaluate, Unit } from "mathjs";
import { type ReactElement, useCallback, useEffect } from "react";

import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { DragButton, type DragButtonExtraProps } from "@/input/DragButton";
import { Text, type TextProps } from "@/input/Text";
import { type Control } from "@/input/types";
import { Triggers } from "@/triggers";

export interface NumericProps
  extends
    Omit<TextProps, "type" | "onBlur" | "value" | "onChange">,
    DragButtonExtraProps,
    Control<number> {
  selectOnFocus?: boolean;
  showDragHandle?: boolean;
  bounds?: bounds.Crude;
  onBlur?: () => void;
  units?: string;
}

/**
 * A controlled number input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @default "medium"
 * @param props.selectOnFocus - Whether the input should select its contents when focused.
 * @default true
 * @param props.centerPlaceholder - Whether the placeholder should be centered.
 * @default false
 * @param props.showDragHandle - Whether or not to show a drag handle to set the time.
 * @default true
 * @param props.dragScale - The scale of the drag handle.
 * @default x: 1, y: 10
 * @param props.dragDirection - The direction of the drag handle.
 * @default undefined
 */
export const Numeric = ({
  ref,
  onChange,
  value,
  dragDirection,
  showDragHandle = true,
  dragScale,
  selectOnFocus = true,
  bounds: propsBounds = bounds.INFINITE,
  resetValue,
  variant = "outlined",
  className,
  children,
  disabled,
  onBlur,
  units,
  size,
  color,
  contrast,
  ...rest
}: NumericProps): ReactElement => {
  // We need to keep the actual value as a valid number, but we need to let the user
  // input an invalid value that may eventually be valid, so we need to keep the
  // internal value as a string in state.
  const [internalValue, setInternalValue, internalValueRef] = useCombinedStateAndRef(
    value.toString(),
  );
  const [isValueValid, setIsValueValid, isValueValidRef] =
    useCombinedStateAndRef<boolean>(true);
  const valueRef = useSyncedRef(value);

  const updateActualValue = useCallback(() => {
    // This just means we never actually modified the input
    if (isValueValidRef.current) return;
    setIsValueValid(true);
    let v = null;
    try {
      const ev = evaluate(internalValueRef.current);
      // Sometimes mathjs returns a Unit object, so we need to convert it to a number.
      if (ev instanceof Unit) v = ev.toNumber();
      else if (typeof ev === "number" && !isNaN(ev)) v = ev;
    } catch {
      v = null;
    }
    if (v != null) onChange?.(bounds.clamp(propsBounds, v));
    else setInternalValue(valueRef.current.toString());
  }, [onChange, setInternalValue]);

  const updateActualValueRef = useSyncedRef(updateActualValue);

  const handleBlur = useCallback(() => {
    onBlur?.();
    updateActualValue();
  }, [onBlur, updateActualValue]);

  // Sometimes we don't blur the component before it unmounts, so this makes
  // sure we try to update the actual value on unmount.
  useEffect(() => () => updateActualValueRef.current?.(), []);

  const handleChange = useCallback(
    (v: string) => {
      setIsValueValid(false);
      setInternalValue(v);
    },
    [setInternalValue, setIsValueValid],
  );

  // If the value is valid, use the actual value, otherwise use the internal value.
  const value_ = isValueValid ? value : internalValue;

  const onDragChange = useCallback(
    (value: number) => {
      setIsValueValid(true);
      onChange?.(bounds.clamp(propsBounds, Math.round(value)));
    },
    [onChange, setIsValueValid],
  );

  if (dragScale == null && bounds.isFinite(propsBounds))
    // make X 5% of the bounds and Y 10% of the bounds
    dragScale = {
      x: bounds.span(propsBounds) * 0.01,
      y: bounds.span(propsBounds) * 0.02,
    };

  if (disabled || variant === "preview") showDragHandle = false;

  return (
    <Text
      ref={ref}
      type="text"
      variant={variant}
      value={value_.toString()}
      onChange={handleChange}
      disabled={disabled}
      selectOnFocus={selectOnFocus}
      // When the user hits 'Enter', we should try to evaluate the input and update the
      // actual value.
      onKeyDown={(e) => {
        if (Triggers.eventKey(e) !== "Enter") return;
        updateActualValue();
        onBlur?.();
      }}
      onBlur={handleBlur}
      size={size}
      color={color}
      contrast={contrast}
      {...rest}
    >
      {showDragHandle && (
        <DragButton
          direction={dragDirection}
          value={value}
          onChange={onDragChange}
          dragScale={dragScale}
          resetValue={resetValue}
          onBlur={handleBlur}
          size={size}
          color={color}
          contrast={contrast}
        />
      )}
      {children}
    </Text>
  );
};

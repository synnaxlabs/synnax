// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, forwardRef, useCallback, useEffect } from "react";

import { bounds } from "@synnaxlabs/x";
import { evaluate } from "mathjs";

import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { DragButton, type DragButtonExtraProps } from "@/input/DragButton";
import { Text, type TextExtraProps } from "@/input/Text";
import { type BaseProps } from "@/input/types";
import { Triggers } from "@/triggers";

export interface NumericProps
  extends Omit<BaseProps<number>, "type" | "onBlur">,
    DragButtonExtraProps,
    TextExtraProps {
  selectOnFocus?: boolean;
  showDragHandle?: boolean;
  bounds?: bounds.Crude;
  onBlur?: () => void;
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
 * @defaul true
 * @param props.centerPlaceholder - Whether the placeholder should be centered.
 * @default false
 * @param props.showDragHandle - Whether or not to show a drag handle to set the time.
 * @default true
 * @param props.dragScale - The scale of the drag handle.
 * @default x: 1, y: 10
 * @param props.dragDirection - The direction of the drag handle.
 * @default undefined
 */
export const Numeric = forwardRef<HTMLInputElement, NumericProps>(
  (
    {
      size = "medium",
      onChange,
      value,
      dragDirection,
      showDragHandle = true,
      dragScale,
      selectOnFocus = true,
      bounds: propsBounds = bounds.INFINITE,
      resetValue,
      style,
      variant = "outlined",
      className,
      children,
      onBlur,
      ...props
    },
    ref,
  ): ReactElement => {
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
      let ok = false;
      let v = 0;
      try {
        v = evaluate(internalValueRef.current);
        ok = v != null;
      } catch (e) {
        ok = false;
      }
      if (ok) onChange?.(bounds.clamp(propsBounds, v));
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

    // We don't communicate the actual value until the user is done dragging, this
    // prevents a bunch of re-renders every time the user moves the mouse.
    const onDragChange = useCallback(
      (value: number) => {
        setIsValueValid(false);
        setInternalValue(Math.round(bounds.clamp(propsBounds, value)).toString());
      },
      [setInternalValue, setIsValueValid],
    );

    // See not above.
    const onDragEnd = useCallback(
      (value: number) => {
        setIsValueValid(true);
        onChange?.(bounds.clamp(propsBounds, Math.round(value)));
      },
      [onChange, setIsValueValid],
    );

    if (dragScale == null && bounds.isFinite(propsBounds)) {
      // make X 5% of the bounds and Y 10% of the bounds
      dragScale = {
        x: bounds.span(propsBounds) * 0.01,
        y: bounds.span(propsBounds) * 0.02,
      };
    }

    return (
      <Text
        ref={ref}
        type="text"
        variant={variant}
        value={value_.toString()}
        onChange={handleChange}
        style={style}
        selectOnFocus={selectOnFocus}
        // When the user hits 'Enter', we should try to evaluate the input and update the
        // actual value.
        onKeyDown={(e) => {
          if (Triggers.eventKey(e) !== "Enter") return;
          updateActualValue();
          onBlur?.();
        }}
        onBlur={handleBlur}
        {...props}
      >
        {showDragHandle && (
          <DragButton
            direction={dragDirection}
            value={value}
            onChange={onDragChange}
            dragScale={dragScale}
            resetValue={resetValue}
            onDragEnd={onDragEnd}
            onBlur={handleBlur}
          />
        )}
        {children}
      </Text>
    );
  },
);
Numeric.displayName = "InputNumber";

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ReactElement,
  forwardRef,
  useCallback,
  useState,
  type FocusEventHandler,
} from "react";

import { bounds } from "@synnaxlabs/x";
import { evaluate } from "mathjs";

import { DragButton, type DragButtonExtensionProps } from "@/input/DragButton";
import { Text } from "@/input/Text";
import { type BaseProps } from "@/input/types";

import { Triggers, useCombinedStateAndRef } from "..";

export interface NumericProps
  extends Omit<BaseProps<number>, "type">,
    DragButtonExtensionProps {
  selectOnFocus?: boolean;
  showDragHandle?: boolean;
  bounds?: bounds.Crude;
}

const toNumber = (v: string | number): [number, boolean] => {
  if (v.toString().length === 0) return [0, false];
  const n = Number(v);
  return [n, !isNaN(n)];
};
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
      bounds: b = bounds.INFINITE,
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
    const [isValueValid, setIsValueValid] = useState(true);

    const updateActualValue = useCallback(() => {
      setIsValueValid(true);
      let ok = false;
      let v = 0;
      try {
        v = evaluate(internalValueRef.current);
        ok = true;
      } catch (e) {
        ok = false;
      }
      if (ok) {
        onChange?.(v);
      } else {
        setInternalValue(value.toString());
      }
    }, [onChange, setInternalValue]);

    const handleBlur: FocusEventHandler<HTMLInputElement> = useCallback(
      (e) => {
        onBlur?.(e);
        updateActualValue();
      },
      [onBlur],
    );

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
        setInternalValue(Math.round(value).toString());
      },
      [setInternalValue, setIsValueValid],
    );

    // See not above.
    const onDragEnd = useCallback(
      (value: number) => {
        setIsValueValid(true);
        onChange?.(value);
      },
      [onChange, setIsValueValid],
    );

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
        onKeyDown={(e) => Triggers.eventKey(e) === "Enter" && updateActualValue()}
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
            onBlur={props.onBlur as FocusEventHandler<HTMLButtonElement>}
          />
        )}
        {children}
      </Text>
    );
  },
);
Numeric.displayName = "InputNumber";

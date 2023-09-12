// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, forwardRef, useCallback, useState } from "react";

import { bounds } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { DragButton, type DragButtonExtensionProps } from "@/input/DragButton";
import { Text } from "@/input/Text";
import { type BaseProps } from "@/input/types";

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
      ...props
    },
    ref,
  ): ReactElement => {
    const [internalValue, setInternalValue] = useState(value.toString());
    const [isValueValid, setIsValueValid] = useState(true);

    const handleChange = useCallback(
      (v: string | number) => {
        let [n, ok] = toNumber(v);
        if (ok) {
          setIsValueValid(true);
          n = bounds.clamp(bounds.construct(b), n);
          setInternalValue(v.toString());
          onChange(n);
        } else {
          setInternalValue(v.toString());
          setIsValueValid(false);
        }
      },
      [setInternalValue, onChange],
    );

    const value_ = isValueValid ? value : internalValue;

    const input = (
      <Text
        ref={ref}
        type="number"
        variant={showDragHandle ? "outlined" : variant}
        value={value_.toString()}
        onChange={handleChange}
        style={showDragHandle ? undefined : style}
        selectOnFocus={selectOnFocus}
        {...props}
      />
    );

    const onDragChange = useCallback(
      (value: number) => handleChange(Math.round(value)),
      [onChange],
    );

    if (!showDragHandle) return input;
    return (
      <Align.Pack
        className={CSS(className, CSS.BM("input", variant), CSS.BE("input", "wrapper"))}
        style={style}
      >
        {input}
        <DragButton
          direction={dragDirection}
          value={value}
          onChange={onDragChange}
          dragScale={dragScale}
          resetValue={resetValue}
        />
      </Align.Pack>
    );
  },
);
Numeric.displayName = "InputNumber";

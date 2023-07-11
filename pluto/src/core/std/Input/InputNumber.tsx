// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, forwardRef, useCallback, useState } from "react";

import { Bounds, CrudeBounds } from "@synnaxlabs/x";

import { Input } from "@/core/std/Input/Input";
import {
  InputDragButton,
  InputDragButtonExtensionProps,
} from "@/core/std/Input/InputDragButton";
import { InputBaseProps } from "@/core/std/Input/types";
import { Pack } from "@/core/std/Pack";

export interface InputNumberProps
  extends Omit<InputBaseProps<number>, "type">,
    InputDragButtonExtensionProps {
  selectOnFocus?: boolean;
  showDragHandle?: boolean;
  bounds?: CrudeBounds;
}

const toNumber = (v: string | number): [number, boolean] => {
  if (v.toString().length === 0) return [0, false];
  const n = Number(v);
  return [n, !isNaN(n)];
};

export const InputNumber = forwardRef<HTMLInputElement, InputNumberProps>(
  (
    {
      size = "medium",
      onChange,
      value,
      dragDirection,
      showDragHandle = true,
      dragScale,
      selectOnFocus = true,
      bounds = Bounds.INFINITE,
      resetValue,
      style,
      ...props
    },
    ref
  ): ReactElement => {
    const [internalValue, setInternalValue] = useState(value.toString());
    const [isValueValid, setIsValueValid] = useState(true);

    const handleChange = useCallback(
      (v: string | number) => {
        let [n, ok] = toNumber(v);
        if (ok) {
          setIsValueValid(true);
          n = new Bounds(bounds).clamp(n);
          setInternalValue(v.toString());
          onChange(n);
        } else {
          setInternalValue(v.toString());
          setIsValueValid(false);
        }
      },
      [setInternalValue, onChange]
    );

    const value_ = isValueValid ? value : internalValue;

    const input = (
      <Input
        ref={ref}
        type="number"
        value={value_.toString()}
        onChange={handleChange}
        style={showDragHandle ? undefined : style}
        selectOnFocus={selectOnFocus}
        {...props}
      />
    );

    const onDragChange = useCallback(
      (value: number) => handleChange(Math.round(value)),
      [onChange]
    );

    if (!showDragHandle) return input;
    return (
      <Pack {...props} style={style}>
        {input}
        <InputDragButton
          direction={dragDirection}
          value={value}
          onChange={onDragChange}
          dragScale={dragScale}
          resetValue={resetValue}
        />
      </Pack>
    );
  }
);
InputNumber.displayName = "InputNumber";

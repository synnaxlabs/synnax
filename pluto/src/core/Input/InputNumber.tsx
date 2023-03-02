// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import { Pack } from "../Pack";

import { Input } from "./Input";
import { InputDragButton, InputDragButtonExtensionProps } from "./InputDragButton";
import { InputBaseProps } from "./types";

import "./InputNumber.css";

export interface InputNumberProps
  extends Omit<InputBaseProps<number>, "type">,
    InputDragButtonExtensionProps {
  selectOnFocus?: boolean;
  showDragHandle?: boolean;
}

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
      ...props
    },
    ref
  ): JSX.Element => {
    const input = (
      <Input
        ref={ref}
        type="number"
        value={String(value) ?? ""}
        onChange={(v: string) => {
          if (v === "") return onChange(NaN);
          onChange(Number(v));
        }}
        selectOnFocus={selectOnFocus}
        {...props}
      />
    );

    if (!showDragHandle) return input;
    return (
      <Pack {...props}>
        {input}
        <InputDragButton direction={dragDirection} value={value} onChange={onChange} />
      </Pack>
    );
  }
);
InputNumber.displayName = "InputNumber";

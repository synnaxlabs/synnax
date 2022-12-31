// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentType, forwardRef } from "react";

import clsx from "clsx";

import { Space } from "../Space";

import { BaseInputProps, Input, InputProps } from "./Input";
import { InputHelpText } from "./InputHelpText";
import { InputLabel } from "./InputLabel";

export interface InputItemProps extends Omit<InputProps, "children"> {
  label?: string;
  helpText?: string;
  children?: ComponentType<BaseInputProps>;
}

export const InputItem = forwardRef<HTMLInputElement, InputItemProps>(
  (
    {
      label,
      helpText,
      style,
      className,
      children: Children = Input,
      ...props
    }: InputItemProps,
    ref
  ) => {
    return (
      <Space
        size="small"
        className={clsx("pluto-input-item", className)}
        direction="vertical"
        style={style}
      >
        <InputLabel>{label}</InputLabel>
        <Children ref={ref} {...props} />
        <InputHelpText>{helpText}</InputHelpText>
      </Space>
    );
  }
);

InputItem.displayName = "InputItem";

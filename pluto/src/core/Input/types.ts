// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DetailedHTMLProps, InputHTMLAttributes } from "react";

import { ComponentSize } from "@/util/component";

export type InputValue = boolean | string | number | readonly string[];

export interface InputControlProps<T extends InputValue = InputValue> {
  value: T;
  onChange: (value: T) => void;
}

type HTMLInputProps = Omit<
  DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>,
  "ref" | "size" | "onChange" | "value" | "children"
>;

export interface InputBaseProps<T extends InputValue = InputValue>
  extends HTMLInputProps,
    InputControlProps<T> {
  size?: ComponentSize;
}

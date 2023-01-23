// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef } from "react";

import { ComponentSize } from "@/util/component";

export type InputValue = boolean | string | number | readonly string[] | null;

export interface InputControl<V extends InputValue = InputValue> {
  value: V;
  onChange: (value: V) => void;
}

type HTMLInputProps = Omit<
  ComponentPropsWithoutRef<"input">,
  "size" | "onChange" | "value" | "children"
>;

export interface InputBaseProps<V extends InputValue = InputValue>
  extends HTMLInputProps,
    InputControl<V> {
  size?: ComponentSize;
}

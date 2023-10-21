// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithoutRef } from "react";

import { type ComponentSize } from "@/util/component";

export type Value = unknown;

export interface Control<I extends Value = Value, O extends Value = I> {
  value: I;
  onChange: (value: O) => void;
}

export interface OptionalControl<I extends Value = Value, O extends Value = I>
  extends Partial<Control<I, O>> {}

type HTMLInputProps = Omit<
  ComponentPropsWithoutRef<"input">,
  "size" | "onChange" | "value" | "children"
>;

export type Variant = "outlined" | "shadow";

export interface BaseProps<I extends Value = Value, O extends Value = I>
  extends HTMLInputProps,
    Control<I, O> {
  size?: ComponentSize;
  variant?: Variant;
  sharp?: boolean;
}

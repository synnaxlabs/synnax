// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithRef } from "react";

export interface Control<I = unknown, O = I> {
  value: I;
  onChange: (value: O) => void;
}

export interface OptionalControl<I = unknown, O = I> extends Partial<Control<I, O>> {}

export type HTMLInputProps = Omit<
  ComponentPropsWithRef<"input">,
  "size" | "onChange" | "value" | "children" | "placeholder" | "color"
>;

export type Variant = "outlined" | "text" | "preview";

export interface InputProps<I = unknown, O = I> extends HTMLInputProps, Control<I, O> {
  variant?: Variant;
}

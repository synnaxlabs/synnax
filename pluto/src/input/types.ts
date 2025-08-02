// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithRef, type ReactNode } from "react";

import { type Button } from "@/button";
import { type Text } from "@/text";

export interface Control<I = unknown, O = I> {
  value: I;
  onChange: (value: O) => void;
}

export interface OptionalControl<I = unknown, O = I> extends Partial<Control<I, O>> {}

type HTMLInputProps = Omit<
  ComponentPropsWithRef<"input">,
  "size" | "onChange" | "value" | "children" | "placeholder" | "color"
>;

export type Variant = "outlined" | "shadow" | "natural" | "preview" | "button";

export interface ExtensionProps<I = unknown, O = I>
  extends Control<I, O>,
    Omit<Button.ExtensionProps, "variant"> {
  variant?: Variant;
  placeholder?: ReactNode;
  children?: ReactNode;
  level?: Text.Level;
  weight?: Text.Weight;
  endContent?: ReactNode;
  onlyChangeOnBlur?: boolean;
}

export interface BaseProps<I = unknown, O = I>
  extends HTMLInputProps,
    ExtensionProps<I, O> {}

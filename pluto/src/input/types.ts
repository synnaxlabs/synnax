// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithoutRef, type ReactNode } from "react";

import { Align } from "@/align";
import { type Text } from "@/text";
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
  "size" | "onChange" | "value" | "children" | "placeholder"
>;

export type Variant = "outlined" | "shadow" | "natural" | "preview";

export interface ExtensionProps<I extends Value = Value, O extends Value = I>
  extends Control<I, O> {
  size?: ComponentSize;
  variant?: Variant;
  sharp?: boolean;
  placeholder?: ReactNode;
  children?: ReactNode;
  level?: Text.Level;
  shade?: Text.Shade;
  weight?: Text.Weight;
  endContent?: ReactNode;
}

export interface BaseProps<I extends Value = Value, O extends Value = I>
  extends HTMLInputProps,
    ExtensionProps<I, O>,
    Pick<Align.PackProps, "borderWidth" | "borderShade"> {}
